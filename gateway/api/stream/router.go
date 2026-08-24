package stream

import "net/http"

func Register(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/streams/", h.stream)
}
