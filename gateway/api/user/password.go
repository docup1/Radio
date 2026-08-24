package user

import "net/http"

// Password godoc
//
//	@Summary	Change password
//	@Tags		auth
//	@Security	bearerAuth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		PasswordUpdate	true	"passwords"
//	@Success	200
//	@Failure	400	{object}	ErrorResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/auth/password [post]
func (h *Handler) password(w http.ResponseWriter, r *http.Request) { h.authProxy(true)(w, r) }
