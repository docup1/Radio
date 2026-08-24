package infra

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewProxy builds a reverse proxy to base, stripping strip (e.g. "/api/auth") from the
// incoming path before forwarding.
func NewProxy(base, strip string) *httputil.ReverseProxy {
	target, _ := url.Parse(base)
	proxy := httputil.NewSingleHostReverseProxy(target)
	orig := proxy.Director
	proxy.Director = func(r *http.Request) {
		orig(r)
		r.URL.Path = strings.TrimPrefix(r.URL.Path, strip)
		if r.URL.RawPath != "" {
			r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, strip)
		}
	}
	return proxy
}

// IsAudioPath reports whether an /api/content sub-path targets the stream service
// (song audio playback).
func IsAudioPath(p string) bool {
	return strings.HasPrefix(p, "/songs/") && strings.HasSuffix(p, "/audio")
}
