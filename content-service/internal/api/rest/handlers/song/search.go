package song

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Search full-text searches the caller's songs.
//
//	@Summary	Search songs
//	@Tags		songs
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		q			query	string	true	"Search query"
//	@Param		limit		query	int		false	"Page limit (default 20, max 100)"
//	@Param		offset		query	int		false	"Page offset"
//	@Success	200			{array}	models.Song
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/songs/search [get]
func Search(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		q := r.URL.Query().Get("q")
		if q == "" {
			common.WriteError(w, http.StatusBadRequest, "missing q parameter")
			return
		}
		limit, offset := common.ParsePagination(r)
		songs, err := svc.Songs.Search(r.Context(), q, owner, limit, offset)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, songs)
	}
}
