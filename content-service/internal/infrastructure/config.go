package infrastructure

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var ErrMissingDBPassword = errors.New("DB_PASSWORD is required")

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

// DSN builds a Postgres connection string for the pgx stdlib driver.
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

// GRPCConfig configures the internal gRPC server (Access.Check).
type GRPCConfig struct {
	Addr string `yaml:"addr"`
}

// StorageConfig configures chunked upload persistence.
type StorageConfig struct {
	// ChunkDir holds raw uploaded chunks, one directory per session.
	ChunkDir string `yaml:"chunk_dir"`
	// FinalDir holds assembled media files.
	FinalDir string `yaml:"final_dir"`
	// MaxChunkSize is the maximum size of a single uploaded chunk in bytes.
	MaxChunkSize int64 `yaml:"max_chunk_size"`
	// MaxFileSize is the maximum size of a fully assembled file in bytes.
	MaxFileSize int64 `yaml:"max_file_size"`
}

// SwaggerConfig toggles the Swagger UI. It is enabled only in dev (see
// SWAGGER_ENABLED) and served on Path (default /swagger) of the HTTP server.
// SpecFile is the path to the generated OpenAPI document (swagger.json /
// swagger.yaml) produced by `make generate`; it is served from disk, never
// embedded or hardcoded.
type SwaggerConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Path      string `yaml:"path"`
	SpecFile  string `yaml:"spec_file"`
}

// Config is the top-level service configuration loaded from a YAML file.
// Values may be overridden per environment (dev/prod) via environment
// variables; see LoadConfig.
type Config struct {
	Env         string        `yaml:"env"`
	LogLevel    string        `yaml:"log_level"`
	HTTPPublic  HTTPConfig    `yaml:"http_public"`
	HTTPPrivate HTTPConfig    `yaml:"http_private"`
	GRPC        GRPCConfig    `yaml:"grpc"`
	DB          DBConfig      `yaml:"db"`
	Storage     StorageConfig `yaml:"storage"`
	Swagger     SwaggerConfig `yaml:"swagger"`

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

	// Sensible defaults for Swagger; overridden by the corresponding env vars.
	if cfg.Swagger.Path == "" {
		cfg.Swagger.Path = "/swagger"
	}
	if cfg.Swagger.SpecFile == "" {
		cfg.Swagger.SpecFile = "docs/swagger.json"
	}

	// Environment overrides. A single base config.yaml is shared across
	// environments; dev/prod differences are supplied via env vars.
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DB.Port = n
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DB.User = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DB.Name = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.DB.SSLMode = v
	}
	if v := os.Getenv("HTTP_PUBLIC_ADDR"); v != "" {
		cfg.HTTPPublic.Addr = v
	}
	if v := os.Getenv("HTTP_PRIVATE_ADDR"); v != "" {
		cfg.HTTPPrivate.Addr = v
	}
	if v := os.Getenv("GRPC_ADDR"); v != "" {
		cfg.GRPC.Addr = v
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
	if v := os.Getenv("STORAGE_CHUNK_DIR"); v != "" {
		cfg.Storage.ChunkDir = v
	}
	if v := os.Getenv("STORAGE_FINAL_DIR"); v != "" {
		cfg.Storage.FinalDir = v
	}

	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	if cfg.DBPassword == "" {
		return nil, ErrMissingDBPassword
	}
	return &cfg, nil
}
