package song

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// Update patches a song's mutable fields.
//
//	@Summary	Update song
//	@Tags		songs
//	@Accept		json
//	@Param		X-Owner-ID	header	string					true	"Owner UUID (set by gateway)"
//	@Param		id			path	string					true	"Song ID"
//	@Param		request		body	dto.UpdateSongRequest	true	"Song patch"
//	@Success	204
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	409			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/songs/{id} [put]
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
		var req dto.UpdateSongRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		patch := interfaces.SongPatch{}
		if req.Name != nil {
			patch.Name = req.Name
		}
		if req.Description != nil {
			patch.Description = req.Description
		}
		if req.MelodyID != nil {
			mid, err := uuid.Parse(*req.MelodyID)
			if err != nil {
				common.WriteError(w, http.StatusBadRequest, "invalid melody_id")
				return
			}
			patch.MelodyID = &mid
		}
		if req.ImageID != nil {
			if *req.ImageID == "" {
				common.WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			iid, err := uuid.Parse(*req.ImageID)
			if err != nil {
				common.WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			patch.ImageID = &uuid.NullUUID{UUID: iid, Valid: true}
		}
		if req.IsPublic != nil {
			patch.IsPublic = req.IsPublic
		}
		if err := svc.Songs.Update(r.Context(), id, owner, patch); err != nil {
			common.WriteServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
