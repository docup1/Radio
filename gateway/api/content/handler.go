package content

import (
	"net/http/httputil"

	"radio/gateway/infra"
)

// Handler proxies the content-service API (REST + stream). Every request is
// authenticated: the gateway validates the token via user-service and sets the
// trusted X-Owner-ID header. Audio is routed to the stream service.
type Handler struct {
	restProxy   *httputil.ReverseProxy
	streamProxy *httputil.ReverseProxy
	auth        *infra.AuthService
}

// New builds a content-service route handler.
func New(restProxy, streamProxy *httputil.ReverseProxy, auth *infra.AuthService) *Handler {
	return &Handler{restProxy: restProxy, streamProxy: streamProxy, auth: auth}
}
