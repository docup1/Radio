package infra

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the gateway configuration. The gateway intentionally holds NO JWT
// secret: it never validates tokens locally. Instead it delegates validation to
// the user-service (calling /me), which checks the session internally.
type Config struct {
	Env       string
	Server    ServerConfig
	Upstreams UpstreamsConfig
	Cookie    CookieConfig
	CORS      CORSConfig
	Swagger   SwaggerConfig
	StaticDir string
}

type ServerConfig struct {
	Addr              string        `yaml:"addr"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
}

type UpstreamsConfig struct {
	UserService    string `yaml:"user_service"`
	ContentService string `yaml:"content_service"`
	ContentStream  string `yaml:"content_stream"`
}

type CookieConfig struct {
	Name         string `yaml:"name"`
	Path         string `yaml:"path"`
	MaxAge       int    `yaml:"max_age"` // seconds
	SecureInProd bool   `yaml:"secure_in_prod"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
}

// SwaggerConfig toggles the gateway's own Swagger UI (the public API contract) and,
// in dev, the proxied upstream service docs. The OpenAPI document is served from
// disk (SpecFile), never embedded.
type SwaggerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	SpecFile string `yaml:"spec_file"`
}

// LoadConfig reads the YAML file and applies environment overrides. A single base
// config.yaml is shared across environments; dev/prod differences are supplied via
// environment variables (see docker-compose.yaml).
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.Env == "" {
		cfg.Env = "dev"
	}
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Server.ReadHeaderTimeout == 0 {
		cfg.Server.ReadHeaderTimeout = 5 * time.Second
	}
	if cfg.Cookie.Name == "" {
		cfg.Cookie.Name = "access_token"
	}
	if cfg.Cookie.Path == "" {
		cfg.Cookie.Path = "/"
	}
	if cfg.Cookie.MaxAge == 0 {
		cfg.Cookie.MaxAge = 86400
	}
	if cfg.Swagger.Path == "" {
		cfg.Swagger.Path = "/swagger"
	}
	if cfg.Swagger.SpecFile == "" {
		cfg.Swagger.SpecFile = "docs/swagger.json"
	}

	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("GATEWAY_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("USER_SERVICE_URL"); v != "" {
		cfg.Upstreams.UserService = v
	}
	if v := os.Getenv("CONTENT_SERVICE_URL"); v != "" {
		cfg.Upstreams.ContentService = v
	}
	if v := os.Getenv("CONTENT_STREAM_URL"); v != "" {
		cfg.Upstreams.ContentStream = v
	}
	if v := os.Getenv("COOKIE_NAME"); v != "" {
		cfg.Cookie.Name = v
	}
	if v := os.Getenv("COOKIE_PATH"); v != "" {
		cfg.Cookie.Path = v
	}
	if v := os.Getenv("COOKIE_MAX_AGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Cookie.MaxAge = n
		}
	}
	if v := os.Getenv("COOKIE_SECURE_IN_PROD"); v != "" {
		cfg.Cookie.SecureInProd = v == "true" || v == "1"
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.CORS.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("SWAGGER_ENABLED"); v != "" {
		cfg.Swagger.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("SWAGGER_PATH"); v != "" {
		cfg.Swagger.Path = v
	}
	if v := os.Getenv("SWAGGER_SPEC_FILE"); v != "" {
		cfg.Swagger.SpecFile = v
	}
	if v := os.Getenv("STATIC_DIR"); v != "" {
		cfg.StaticDir = v
	}

	if cfg.Upstreams.UserService == "" {
		return nil, errors.New("user_service upstream URL is required")
	}
	if cfg.Upstreams.ContentService == "" {
		return nil, errors.New("content_service upstream URL is required")
	}
	if cfg.Upstreams.ContentStream == "" {
		return nil, errors.New("content_stream upstream URL is required")
	}
	return &cfg, nil
}
