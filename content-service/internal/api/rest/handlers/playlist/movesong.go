package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// MoveSong reorders a song within a playlist.
//
//	@Summary	Reorder playlist song
//	@Tags		playlists
//	@Accept		json
//	@Param		X-Owner-ID	header	string				true	"Owner UUID (set by gateway)"
//	@Param		id			path	string				true	"Playlist ID"
//	@Param		song_id		path	string				true	"Song ID"
//	@Param		request		body	dto.MoveSongRequest	true	"Move payload"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists/{id}/songs/{song_id} [put]
func MoveSong(svc *application.Services) http.HandlerFunc {
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
		songID, err := common.ParseID(r, "song_id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req dto.MoveSongRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := svc.PlaylistSongs.Move(r.Context(), playlistID, owner, songID, req.Position); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
