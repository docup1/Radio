package song

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// Create creates a new song owned by the caller.
//
//	@Summary	Create song
//	@Tags		songs
//	@Accept		json
//	@Produce	json
//	@Param		X-Owner-ID	header	string					true	"Owner UUID (set by gateway)"
//	@Param		request		body	dto.CreateSongRequest	true	"Song payload"
//	@Success	201			{object}	models.Song
//	@Failure	400			{object}	map[string]string
//	@Failure	404			{object}	map[string]string
//	@Failure	409			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/songs [post]
func Create(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		var req dto.CreateSongRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		mid, err := uuid.Parse(req.MelodyID)
		if err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid melody_id")
			return
		}
		var imgID *uuid.UUID
		if req.ImageID != nil {
			id, err := uuid.Parse(*req.ImageID)
			if err != nil {
				common.WriteError(w, http.StatusBadRequest, "invalid image_id")
				return
			}
			imgID = &id
		}
		s, err := svc.Songs.Create(r.Context(), owner, application.CreateSongInput{
			Name:        req.Name,
			Description: req.Description,
			MelodyID:    mid,
			ImageID:     imgID,
			IsPublic:    req.IsPublic,
		})
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, s)
	}
}
