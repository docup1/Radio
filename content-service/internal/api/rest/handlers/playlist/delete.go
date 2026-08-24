package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Delete removes a playlist.
//
//	@Summary	Delete playlist
//	@Tags		playlists
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Playlist ID"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists/{id} [delete]
func Delete(svc *application.Services) http.HandlerFunc {
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
		if err := svc.Playlists.Delete(r.Context(), id, owner); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
