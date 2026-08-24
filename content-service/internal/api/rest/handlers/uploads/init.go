package uploads

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// Init starts a chunked upload session.
//
//	@Summary	Init upload
//	@Tags		uploads
//	@Accept		json
//	@Produce	json
//	@Param		X-Owner-ID	header	string					true	"Owner UUID (set by gateway)"
//	@Param		request		body	dto.InitUploadRequest		true	"Upload session payload"
//	@Success	201			{object}	models.UploadSession
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/uploads [post]
func Init(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		var req dto.InitUploadRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		session, err := svc.Uploads.Init(r.Context(), owner, application.InitUploadInput{
			MediaType:    req.MediaType,
			ContentType:  req.ContentType,
			TotalChunks:  req.TotalChunks,
			ExpectedSize: req.ExpectedSize,
			ExpectedHash: req.ExpectedHash,
		})
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, session)
	}
}
