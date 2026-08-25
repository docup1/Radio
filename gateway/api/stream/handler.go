package stream

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"radio/gateway/infra"
)

type Handler struct {
	proxy   *httputil.ReverseProxy
	auth    *infra.AuthService
	wsProxy *infra.WSProxy
}

func New(proxy *httputil.ReverseProxy, auth *infra.AuthService, ws *infra.WSProxy) *Handler {
	return &Handler{proxy: proxy, auth: auth, wsProxy: ws}
}

// ws handles WebSocket connections: /api/streams/{id}/ws
// Extracts stream ID from path and proxies to sender-service.
func (h *Handler) ws(w http.ResponseWriter, r *http.Request) {
	if h.wsProxy == nil {
		infra.WriteError(w, http.StatusBadGateway, "sender service not configured")
		return
	}

	// Extract stream ID from path: /api/streams/{id}/ws
	path := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	streamID := strings.TrimSuffix(path, "/ws")

	if streamID == "" {
		infra.WriteError(w, http.StatusBadRequest, "missing stream id")
		return
	}

	h.wsProxy.ServeWS(w, r, "/stream/"+streamID)
}
