package infrastructure

import (
	"errors"
	"fmt"
	"net/url"
	"os"
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

// Config is the top-level service configuration loaded from a YAML file.
type Config struct {
	HTTPPublic  HTTPConfig   `yaml:"http_public"`
	HTTPPrivate HTTPConfig   `yaml:"http_private"`
	GRPC        GRPCConfig   `yaml:"grpc"`
	DB          DBConfig     `yaml:"db"`
	Storage     StorageConfig `yaml:"storage"`

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

	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	if cfg.DBPassword == "" {
		return nil, ErrMissingDBPassword
	}
	return &cfg, nil
}
