package api

import (
	"net/http"
	"time"

	"radio/gateway/api/content"
	"radio/gateway/api/user"
	"radio/gateway/infra"
)

// NewRouter builds the gateway mux: health, user routes, content routes, and (in dev)
// Swagger. Auth cookie handling is wired into the user-service proxy via
// ModifyResponse. The gateway holds no JWT secret.
func NewRouter(cfg *infra.Config) http.Handler {
	client := &http.Client{Timeout: 30 * time.Second}
	authSvc := infra.NewAuthService(client, cfg.Upstreams.UserService, cfg.Cookie, cfg.Env)

	userProxy := infra.NewProxy(cfg.Upstreams.UserService, "/api/auth")
	userProxy.ModifyResponse = func(res *http.Response) error {
		switch res.Request.URL.Path {
		case "/register", "/login":
			return authSvc.SetCookieFromBody(res)
		case "/logout":
			authSvc.ClearCookie(res)
		}
		return nil
	}

	restProxy := infra.NewProxy(cfg.Upstreams.ContentService, "/api/content")
	streamProxy := infra.NewProxy(cfg.Upstreams.ContentStream, "/api/content")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	user.Register(mux, user.New(userProxy, authSvc))
	content.Register(mux, content.New(restProxy, streamProxy, authSvc))
	infra.Mount(mux, cfg)

	if cfg.StaticDir != "" {
		mux.Handle("/", infra.StaticHandler(cfg.StaticDir))
	}

	return infra.Middleware(cfg, mux)
}
