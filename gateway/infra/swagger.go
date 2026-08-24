package infra

import (
	"net/http"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Mount serves the gateway's own Swagger UI (the public contract) when enabled, and
// — in dev only — also proxies the upstream service Swagger UIs so developers can
// read the full backend schemas without leaving the gateway.
func Mount(mux *http.ServeMux, cfg *Config) {
	if !cfg.Swagger.Enabled {
		return
	}

	path := cfg.Swagger.Path
	specURL := path + "/swagger.json"
	mux.HandleFunc("GET "+specURL, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, cfg.Swagger.SpecFile)
	})
	mux.Handle(path+"/", httpSwagger.Handler(func(c *httpSwagger.Config) {
		c.URL = specURL
	}))

	if cfg.Env != "dev" {
		return
	}
	mountUpstream(mux, "user-service", cfg.Upstreams.UserService)
	mountUpstream(mux, "content-service", cfg.Upstreams.ContentService)
}

// mountUpstream exposes an upstream service's entire Swagger UI under
// /dev-docs/<name>/ by proxying its /swagger path. The upstream serves a working UI
// (assets + spec) itself, so we only rewrite the path prefix. The paths shown in
// that spec are the service's own (e.g. /songs), not the gateway-prefixed ones.
func mountUpstream(mux *http.ServeMux, name, upstreamBase string) {
	base := "/dev-docs/" + name
	target := strings.TrimRight(upstreamBase, "/") + "/swagger"
	mux.Handle(base+"/", NewProxy(target, base))
}
