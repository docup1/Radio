package playlist

import (
	"net/http"

	"radio/content-service/internal/api/rest/handlers/common"
	"radio/content-service/internal/api/rest/handlers/dto"
	"radio/content-service/internal/application"
)

// Create creates a new playlist owned by the caller.
//
//	@Summary	Create playlist
//	@Tags		playlists
//	@Accept		json
//	@Produce	json
//	@Param		X-Owner-ID	header	string						true	"Owner UUID (set by gateway)"
//	@Param		request		body	dto.CreatePlaylistRequest	true	"Playlist payload"
//	@Success	201			{object}	models.Playlist
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/playlists [post]
func Create(svc *application.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		owner, ok := common.OwnerOrError(w, r)
		if !ok {
			return
		}
		var req dto.CreatePlaylistRequest
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		p, err := svc.Playlists.Create(r.Context(), owner, application.CreatePlaylistInput{Name: req.Name})
		if err != nil {
			common.WriteServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, p)
	}
}
