package user

import "net/http"

// Register godoc
//
//	@Summary	Register a new user (auto-login)
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		request	body		Credentials	true	"credentials"
//	@Success	201	{object}	AuthResponse
//	@Failure	400	{object}	ErrorResponse
//	@Router		/api/auth/register [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) { h.authProxy(false)(w, r) }
