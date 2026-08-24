package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// List returns the caller's playlists (paginated).
//
//	@Summary	List playlists
//	@Tags		playlists
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		limit		query	int		false	"Page limit (default 20, max 100)"
//	@Param		offset		query	int		false	"Page offset"
//	@Success	200			{array}	models.Playlist
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists [get]
func List(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		limit, offset := common.ParsePagination(r)
		out, err := svc.Playlists.List(r.Context(), owner, limit, offset)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, out)
	}
}
