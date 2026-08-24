package user

import "net/http"

// Login godoc
//
//	@Summary	Log in
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		Credentials	true	"credentials"
//	@Success	200	{object}	AuthResponse
//	@Failure	400	{object}	ErrorResponse
//	@Failure	401	{object}	ErrorResponse
//	@Router		/api/auth/login [post]
func (h *Handler) login(w http.ResponseWriter, r *http.Request) { h.authProxy(false)(w, r) }
