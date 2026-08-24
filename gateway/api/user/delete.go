package user

import "net/http"

// Delete godoc
//
//	@Summary	Delete account
//	@Tags		auth
//	@Security	bearerAuth
//	@Success	204
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/auth/me [delete]
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) { h.authProxy(true)(w, r) }
