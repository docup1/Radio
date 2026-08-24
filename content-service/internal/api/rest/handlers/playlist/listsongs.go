package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// ListSongs returns the songs of a playlist in playback order (by position).
//
//	@Summary	List playlist songs
//	@Tags		playlists
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Playlist ID"
//	@Success	200			{array}	models.Song
//	@Failure	400			{object}	map[string]string
//	@Failure	403			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Router		/playlists/{id}/songs [get]
func ListSongs(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		songs, err := svc.PlaylistSongs.List(r.Context(), id, owner)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, songs)
	}
}
