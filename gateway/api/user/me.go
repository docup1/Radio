package user

import "net/http"

// Me godoc
//
//	@Summary	Current user
//	@Tags		auth
//	@Security	bearerAuth
//	@Produce	json
//	@Success	200	{object}	AuthResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/auth/me [get]
func (h *Handler) me(w http.ResponseWriter, r *http.Request) { h.authProxy(true)(w, r) }
