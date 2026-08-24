package playlist

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// AddSong adds a song to a playlist.
//
//	@Summary	Add song to playlist
//	@Tags		playlists
//	@Accept		json
//	@Param		X-Owner-ID	header	string				true	"Owner UUID (set by gateway)"
//	@Param		id			path	string				true	"Playlist ID"
//	@Param		request		body	dto.AddSongRequest	true	"Add song payload"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	409			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists/{id}/songs [post]
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
		var req dto.AddSongRequest
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
