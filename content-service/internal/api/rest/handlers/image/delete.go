package image

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Delete removes an image.
//
//	@Summary	Delete image
//	@Tags		images
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Image ID"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/images/{id} [delete]
func Delete(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Images.Delete(r.Context(), id); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
