package song

import (
	"net/http"

	"github.com/google/uuid"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/application"
)

func Create(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		var req struct {
			Name        string  `json:"name"`
			Description *string `json:"description"`
			MelodyID    string  `json:"melody_id"`
			ImageID     *string `json:"image_id"`
			IsPublic    bool    `json:"is_public"`
		}
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
