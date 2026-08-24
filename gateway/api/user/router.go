package user

import "net/http"

// Register wires the user-service routes onto the mux. Add a new route by appending a
// HandleFunc call here and writing its handler file (with a // @Router annotation).
func Register(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("POST /api/auth/password", h.password)
	mux.HandleFunc("DELETE /api/auth/me", h.delete)
}
