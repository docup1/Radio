package uploads

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Confirm assembles the uploaded chunks into a melody (audio) or image (cover).
//
//	@Summary	Confirm upload
//	@Tags		uploads
//	@Produce	json
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Upload session ID"
//	@Success	201			{object}	models.Melody
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	409			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/uploads/{id}/confirm [post]
func Confirm(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}
		res, err := svc.Uploads.Confirm(r.Context(), owner, id)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		if res.Image != nil {
			common.WriteJSON(w, http.StatusCreated, res.Image)
			return
		}
		common.WriteJSON(w, http.StatusCreated, res.Melody)
	}
}
