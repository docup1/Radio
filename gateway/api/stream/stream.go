package stream

import (
	"net/http"
	"strings"

	"radio/gateway/infra"
)

func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	// /api/streams/{id}/ws → WebSocket proxy (no auth)
	if strings.HasSuffix(r.URL.Path, "/ws") {
		h.ws(w, r)
		return
	}
	// Everything else → REST proxy (auth required)
	h.stream(w, r)
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("X-Owner-ID")
	r.Header.Del("Authorization")

	token := h.auth.ExtractToken(r)
	if token == "" {
		infra.WriteError(w, http.StatusUnauthorized, "missing authentication")
		return
	}
	uid, err := h.auth.Validate(r.Context(), token)
	if err != nil {
		infra.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	r.Header.Set("X-Owner-ID", uid)
	h.proxy.ServeHTTP(w, r)
}
