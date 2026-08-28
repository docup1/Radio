package stream

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"radio/gateway/infra"
)

type Handler struct {
	proxy      *httputil.ReverseProxy
	auth       *infra.AuthService
	wsProxy    *infra.WSProxy
	senderProxy *infra.SenderProxy
}

func New(proxy *httputil.ReverseProxy, auth *infra.AuthService, ws *infra.WSProxy, sender *infra.SenderProxy) *Handler {
	return &Handler{proxy: proxy, auth: auth, wsProxy: ws, senderProxy: sender}
}

// route dispatches /api/streams/*:
//   - {id}/ws → sender-service WebSocket proxy (no auth)
//   - {id}/skip → sender-service REST skip endpoint (auth required)
//   - всё остальное → stream-service REST (auth required)
func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/ws") {
		h.ws(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/skip") {
		h.skip(w, r)
		return
	}
	h.stream(w, r)
}

// ws proxies WebSocket connections to sender-service.
func (h *Handler) ws(w http.ResponseWriter, r *http.Request) {
	if h.wsProxy == nil {
		infra.WriteError(w, http.StatusBadGateway, "sender service not configured")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	streamID := strings.TrimSuffix(path, "/ws")
	if streamID == "" {
		infra.WriteError(w, http.StatusBadRequest, "missing stream id")
		return
	}

	h.wsProxy.ServeWS(w, r, "/stream/"+streamID)
}

// skip proxies skip requests to sender-service with auth.
func (h *Handler) skip(w http.ResponseWriter, r *http.Request) {
	if h.senderProxy == nil {
		infra.WriteError(w, http.StatusBadGateway, "sender service not configured")
		return
	}

	token := h.auth.ExtractToken(r)
	r.Header.Del("X-Owner-ID")
	r.Header.Del("Authorization")
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
	h.senderProxy.ServeHTTP(w, r)
}

// stream proxies control requests to stream-service with auth.
func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	token := h.auth.ExtractToken(r)

	r.Header.Del("X-Owner-ID")
	r.Header.Del("Authorization")
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
