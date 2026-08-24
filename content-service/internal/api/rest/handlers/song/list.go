package song

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/models"
)

// List returns the caller's songs (paginated).
//
//	@Summary	List songs
//	@Tags		songs
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		scope		query	string	false	"mine (default) | public"
//	@Param		limit		query	int		false	"Page limit (default 20, max 100)"
//	@Param		offset		query	int		false	"Page offset"
//	@Success	200			{array}	models.Song
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/songs [get]
func List(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		scope := models.ParseSongScope(r.URL.Query().Get("scope"))
		limit, offset := common.ParsePagination(r)
		songs, err := svc.Songs.List(r.Context(), owner, scope, limit, offset)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, songs)
	}
}
