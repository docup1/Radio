package infra

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// SenderProxy is a reverse proxy to sender-service for REST endpoints (skip).
type SenderProxy struct {
	proxy *httputil.ReverseProxy
}

// NewSenderProxy creates a proxy to sender-service HTTP endpoint.
func NewSenderProxy(baseURL string) *SenderProxy {
	target, _ := url.Parse(baseURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	orig := proxy.Director
	proxy.Director = func(r *http.Request) {
		orig(r)
		r.URL.Path = "/stream" + strings.TrimPrefix(r.URL.Path, "/api/streams")
		if r.URL.RawPath != "" {
			r.URL.RawPath = "/stream" + strings.TrimPrefix(r.URL.RawPath, "/api/streams")
		}
	}
	return &SenderProxy{proxy: proxy}
}

func (p *SenderProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
