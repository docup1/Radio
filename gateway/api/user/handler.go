package user

import (
	"net/http"
	"net/http/httputil"

	"radio/gateway/infra"
)

// Handler proxies the user-service auth API. The token returned by register/login is
// moved into an HttpOnly cookie by the proxy's ModifyResponse (configured in the
// gateway router). me/logout/password forward the caller's token as the Authorization
// header.
type Handler struct {
	proxy *httputil.ReverseProxy
	auth  *infra.AuthService
}

// New builds a user-service route handler.
func New(proxy *httputil.ReverseProxy, auth *infra.AuthService) *Handler {
	return &Handler{proxy: proxy, auth: auth}
}

// authProxy proxies to user-service. When require is true the caller must supply a
// token (cookie or Bearer), forwarded as the Authorization header; client-supplied
// X-Owner-ID is always stripped.
func (h *Handler) authProxy(require bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-Owner-ID")
		if require {
			token := h.auth.ExtractToken(r)
			if token == "" {
				infra.WriteError(w, http.StatusUnauthorized, "missing authentication")
				return
			}
			r.Header.Set("Authorization", "Bearer "+token)
		}
		h.proxy.ServeHTTP(w, r)
	}
}
