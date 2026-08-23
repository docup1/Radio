package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

func Update(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		id, err := common.ParseID(r, "id")
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		var req struct {
			Name     *string `json:"name"`
			IsPublic *bool   `json:"is_public"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.PlaylistPatch{}
		if req.Name != nil {
			patch.Name = req.Name
		}
		if req.IsPublic != nil {
			patch.IsPublic = req.IsPublic
		}
		if err := svc.Playlists.Update(r.Context(), id, owner, patch); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
