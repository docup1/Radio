package playlist

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

func AddSong(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		playlistID, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			SongID   string `json:"song_id"`
			Position *int   `json:"position"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sid, err := uuid.Parse(req.SongID)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid song_id")
			return
		}
		if err := svc.PlaylistSongs.Add(r.Context(), playlistID, owner, application.AddSongInput{SongID: sid, Position: req.Position}); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
