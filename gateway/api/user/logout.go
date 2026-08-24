package user

import "net/http"

// Logout godoc
//
//	@Summary	Log out
//	@Tags		auth
//	@Security	bearerAuth
//	@Success	204
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/auth/logout [post]
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) { h.authProxy(true)(w, r) }
