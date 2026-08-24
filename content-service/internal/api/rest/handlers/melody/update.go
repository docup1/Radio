package melody

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// Update patches a melody's mutable fields.
//
//	@Summary	Update melody
//	@Tags		melodies
//	@Accept		json
//	@Param		X-Owner-ID	header	string						true	"Owner UUID (set by gateway)"
//	@Param		id			path	string						true	"Melody ID"
//	@Param		request		body	dto.UpdateMelodyRequest		true	"Melody patch"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/melodies/{id} [put]
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
		var req dto.UpdateMelodyRequest
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
