package infra

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrMissingJWTSecret  = errors.New("JWT_SECRET is required")
	ErrMissingDBPassword = errors.New("DB_PASSWORD is required")
)

type HTTPConfig struct {
	Addr              string        `yaml:"addr"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
}

type DBConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Name            string        `yaml:"name"`
	SSLMode         string        `yaml:"sslmode"`
	PingTimeout     time.Duration `yaml:"ping_timeout"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

func (c DBConfig) DSN(password string) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Name,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

type AuthConfig struct {
	TokenTTL        time.Duration `yaml:"token_ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

type ValidationConfig struct {
	UsernameMinLength int `yaml:"username_min_length"`
	UsernameMaxLength int `yaml:"username_max_length"`
	PasswordMinLength int `yaml:"password_min_length"`
	PasswordMaxLength int `yaml:"password_max_length"`
}

func (v ValidationConfig) ValidPassword(password string) bool {
	return len(password) >= v.PasswordMinLength && len(password) <= v.PasswordMaxLength
}

type BcryptConfig struct {
	Cost          int `yaml:"cost"`
	MaxConcurrent int `yaml:"max_concurrent"`
}

// SwaggerConfig toggles the Swagger UI. It is enabled only in dev (see
// SWAGGER_ENABLED) and served on Path (default /swagger) of the HTTP server.
// SpecFile is the path to the generated OpenAPI document (swagger.json /
// swagger.yaml) produced by `make generate`; it is served from disk, never
// embedded or hardcoded.
type SwaggerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	SpecFile string `yaml:"spec_file"`
}

type Config struct {
	HTTP       HTTPConfig       `yaml:"http"`
	DB         DBConfig         `yaml:"db"`
	Auth       AuthConfig       `yaml:"auth"`
	Validation ValidationConfig `yaml:"validation"`
	Bcrypt     BcryptConfig     `yaml:"bcrypt"`
	Swagger    SwaggerConfig    `yaml:"swagger"`

	JWTSecret  []byte
	DBPassword string
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.Swagger.Path == "" {
		cfg.Swagger.Path = "/swagger"
	}
	if cfg.Swagger.SpecFile == "" {
		cfg.Swagger.SpecFile = "docs/swagger.json"
	}

	cfg.JWTSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(cfg.JWTSecret) == 0 {
		return nil, ErrMissingJWTSecret
	}
	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	if cfg.DBPassword == "" {
		return nil, ErrMissingDBPassword
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
	return &cfg, nil
}
