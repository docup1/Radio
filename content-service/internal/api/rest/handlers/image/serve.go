package image

import (
	"net/http"
	"os"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// ServeFile returns the raw image bytes for a cover by ID. Any authenticated
// caller may fetch an image, because covers may be referenced by public songs.
//
//	@Summary	Serve image file
//	@Tags		images
//	@Produce	application/octet-stream
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Image ID"
//	@Success	200
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Router		/images/{id}/file [get]
func ServeFile(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		img, err := svc.Images.Get(r.Context(), id)
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		data, err := os.ReadFile(img.Path)
		if err != nil {
			common.WriteError(w, http.StatusNotFound, "file not found")
			return
		}
		w.Header().Set("Content-Type", img.ContentType)
		w.Header().Set("Cache-Control", "private, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}
