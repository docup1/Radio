package image

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// Create registers a new image reference.
//
//	@Summary	Create image
//	@Tags		images
//	@Accept		json
//	@Produce	json
//	@Param		X-Owner-ID	header	string					true	"Owner UUID (set by gateway)"
//	@Param		request		body	dto.CreateImageRequest	true	"Image payload"
//	@Success	201			{object}	models.Image
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/images [post]
func Create(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		var req dto.CreateImageRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		m, err := svc.Images.Create(r.Context(), application.CreateFileInput{Path: req.Path, ContentType: req.ContentType})
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, m)
	}
}
