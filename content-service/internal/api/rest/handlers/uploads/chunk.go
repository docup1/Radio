package uploads

import (
	"io"
	"net/http"
	"strconv"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

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
