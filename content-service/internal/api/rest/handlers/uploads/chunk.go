package uploads

import (
	"io"
	"net/http"
	"strconv"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

// Chunk uploads a single binary chunk of an upload session.
//
//	@Summary	Upload chunk
//	@Tags		uploads
//	@Accept		application/octet-stream
//	@Param		X-Owner-ID	header	string	true	"Owner UUID (set by gateway)"
//	@Param		id			path	string	true	"Upload session ID"
//	@Param		index		path	int		true	"Chunk index"
//	@Param		chunk		body	string	true	"Raw chunk bytes"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/uploads/{id}/chunks/{index} [put]
func Chunk(svc *application.Services) http.HandlerFunc {
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
		index, err := strconv.Atoi(r.PathValue("index"))
		if err != nil || index < 0 {
			common.WriteError(w, http.StatusBadRequest, "invalid index")
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "cannot read body")
			return
		}
		err = svc.Uploads.AddChunk(r.Context(), owner, id, application.AddChunkInput{
			Index: index,
			Data:  data,
		})
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
