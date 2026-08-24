package content

import "net/http"

// Register wires the content-service routes onto the mux. Add a new route by appending
// a HandleFunc call here and writing its handler (with a // @Router annotation).
func Register(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/content/", h.content)
}
