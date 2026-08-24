package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// RemoveSong removes a song from a playlist.
//
//	@Summary	Remove song from playlist
//	@Tags		playlists
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Playlist ID"
//	@Param		song_id		path	string	true	"Song ID"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists/{id}/songs/{song_id} [delete]
func RemoveSong(svc *application.Services) http.HandlerFunc {
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
		if err := svc.PlaylistSongs.Remove(r.Context(), playlistID, owner, songID); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
