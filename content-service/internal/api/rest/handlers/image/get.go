package image

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Get returns a single image by ID.
//
//	@Summary	Get image
//	@Tags		images
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Image ID"
//	@Success	200			{object}	models.Image
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/images/{id} [get]
func Get(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		m, err := svc.Images.Get(r.Context(), id)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, m)
	}
}
