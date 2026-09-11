package infra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const gatewayTestYAML = `---
env: test
server:
  addr: ":9999"
  read_header_timeout: 3s
upstreams:
  user_service: "http://user:8080"
  content_service: "http://content:8080"
  content_stream: "http://content-stream:8080"
  stream_service: "http://stream:8080"
  sender_service: "ws://sender:8081/ws"
cookie:
  name: sid
  path: /
  max_age: 3600
  secure_in_prod: false
cors:
  allowed_origins: ["http://localhost:5173"]
  allowed_methods: ["GET", "POST"]
swagger:
  enabled: true
  path: /docs
  spec_file: docs/api.json
static_dir: /var/www
`

func writeGatewayConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Defaults(t *testing.T) {
	path := writeGatewayConfig(t, `---
upstreams:
  user_service: "http://user:8080"
  content_service: "http://content:8080"
  content_stream: "http://cs:8080"
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "dev" {
		t.Fatalf("env = %q, want dev", cfg.Env)
	}
	if cfg.Server.Addr != ":8080" {
		t.Fatalf("addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Server.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Cookie.Name != "access_token" || cfg.Cookie.Path != "/" || cfg.Cookie.MaxAge != 86400 {
		t.Fatalf("cookie defaults = %+v", cfg.Cookie)
	}
	if cfg.Swagger.Path != "/swagger" || cfg.Swagger.SpecFile != "docs/swagger.json" {
		t.Fatalf("swagger defaults = %+v", cfg.Swagger)
	}
}

func TestLoadConfig_ReadsValues(t *testing.T) {
	path := writeGatewayConfig(t, gatewayTestYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "test" || cfg.Server.Addr != ":9999" {
		t.Fatalf("load = env %q addr %q", cfg.Env, cfg.Server.Addr)
	}
	if cfg.Upstreams.UserService != "http://user:8080" ||
		cfg.Upstreams.ContentStream != "http://content-stream:8080" ||
		cfg.Upstreams.SenderService != "ws://sender:8081/ws" {
		t.Fatalf("upstreams = %+v", cfg.Upstreams)
	}
	if cfg.Cookie.Name != "sid" || cfg.Swagger.Path != "/docs" {
		t.Fatalf("values = %+v", cfg)
	}
	if cfg.StaticDir != "" {
		// StaticDir has no yaml tag; it is env-driven only (STATIC_DIR).
		t.Fatalf("static from yaml = %q, want empty (env-only)", cfg.StaticDir)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("GATEWAY_ADDR", ":7777")
	t.Setenv("USER_SERVICE_URL", "http://env-user:1111")
	t.Setenv("COOKIE_NAME", "env_cookie")
	t.Setenv("COOKIE_MAX_AGE", "42")
	t.Setenv("COOKIE_SECURE_IN_PROD", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a,http://b")
	t.Setenv("SWAGGER_ENABLED", "1")
	t.Setenv("SWAGGER_PATH", "/envdocs")
	t.Setenv("STATIC_DIR", "/env/www")

	path := writeGatewayConfig(t, gatewayTestYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "prod" || cfg.Server.Addr != ":7777" {
		t.Fatalf("env overrides = env %q addr %q", cfg.Env, cfg.Server.Addr)
	}
	if cfg.Upstreams.UserService != "http://env-user:1111" {
		t.Fatalf("user override = %q", cfg.Upstreams.UserService)
	}
	if cfg.Cookie.Name != "env_cookie" || cfg.Cookie.MaxAge != 42 || !cfg.Cookie.SecureInProd {
		t.Fatalf("cookie overrides = %+v", cfg.Cookie)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 || cfg.CORS.AllowedOrigins[1] != "http://b" {
		t.Fatalf("cors = %v", cfg.CORS.AllowedOrigins)
	}
	if !cfg.Swagger.Enabled || cfg.Swagger.Path != "/envdocs" {
		t.Fatalf("swagger = %+v", cfg.Swagger)
	}
	if cfg.StaticDir != "/env/www" {
		t.Fatalf("static = %+v", cfg.StaticDir)
	}
}

func TestLoadConfig_MissingUpstreams(t *testing.T) {
	path := writeGatewayConfig(t, "---\nupstreams:\n  user_service: http://u:1\n  content_service: http://c:1\n")
	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "content_stream") {
		t.Fatalf("err = %v, want content_stream required", err)
	}

	path = writeGatewayConfig(t, "---\nupstreams:\n  content_service: http://c:1\n  content_stream: http://cs:1\n")
	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "user_service") {
		t.Fatalf("err = %v, want user_service required", err)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := LoadConfig(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("want error for missing file")
	}
}
