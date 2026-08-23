package melody

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

func Update(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := common.OwnerOrError(w, r); !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Path        *string `json:"path"`
			ContentType *string `json:"content_type"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.MelodyPatch{}
		if req.Path != nil {
			patch.Path = req.Path
		}
		if req.ContentType != nil {
			patch.ContentType = req.ContentType
		}
		if err := svc.Melodies.Update(r.Context(), id, patch); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
