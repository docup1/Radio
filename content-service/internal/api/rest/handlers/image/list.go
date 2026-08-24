package image

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// List returns the caller's images (paginated).
//
//	@Summary	List images
//	@Tags		images
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		limit		query	int		false	"Page limit (default 20, max 100)"
//	@Param		offset		query	int		false	"Page offset"
//	@Success	200			{array}	models.Image
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/images [get]
func List(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		limit, offset := common.ParsePagination(r)
		out, err := svc.Images.List(r.Context(), limit, offset)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, out)
	}
}
