package uploads

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

func Init(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		var req struct {
			MediaType    string `json:"media_type"`
			ContentType  string `json:"content_type"`
			TotalChunks  int    `json:"total_chunks"`
			ExpectedSize int64  `json:"expected_size"`
			ExpectedHash string `json:"expected_hash"`
		}
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
