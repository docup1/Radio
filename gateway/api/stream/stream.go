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
//   - GET → stream-service read-only (no auth; search/list are public)
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
	if r.Method == http.MethodGet {
		h.read(w, r)
		return
	}
	h.stream(w, r)
}

// read proxies read-only GET requests to stream-service. Public: search/list,
// stream info, state, queue, hashtags work anonymously. When a valid token is
// present the owner is still stamped so e.g. GET /api/streams returns the
// caller's own stream.
func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	r.Header.Del("X-Owner-ID")
	r.Header.Del("Authorization")
	if token := h.auth.ExtractToken(r); token != "" {
		if uid, err := h.auth.Validate(r.Context(), token); err == nil {
			r.Header.Set("X-Owner-ID", uid)
		}
	}
	h.proxy.ServeHTTP(w, r)
}

// ws proxies WebSocket connections to sender-service. The owner is
// authenticated at the gateway: if the session token matches the stream owner
// (stream ID == owner user ID) an X-Owner-ID header is stamped on the upstream
// dial so sender-service can authorize control commands.
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

	var dialHeaders http.Header
	if token := h.auth.ExtractToken(r); token != "" {
		if uid, err := h.auth.Validate(r.Context(), token); err == nil && uid == streamID {
			dialHeaders = http.Header{"X-Owner-ID": []string{uid}}
		}
	}

	h.wsProxy.ServeWS(w, r, "/stream/"+streamID, dialHeaders)
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
