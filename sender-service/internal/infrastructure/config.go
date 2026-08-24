package infrastructure

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

var ErrMissingDBPassword = errors.New("DB_PASSWORD is required")

type HTTPConfig struct {
	Addr              string        `yaml:"addr"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Config struct {
	Env               string     `yaml:"env"`
	LogLevel          string     `yaml:"log_level"`
	HTTP              HTTPConfig `yaml:"http"`
	Redis             RedisConfig `yaml:"redis"`
	StreamServiceGRPC string     `yaml:"stream_service_grpc"`
	ContentServiceURL string     `yaml:"content_service_url"`
	ChunkSize         int64      `yaml:"chunk_size"`
	BufferSeconds     int        `yaml:"buffer_seconds"`
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

	// defaults
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8081"
	}
	if cfg.HTTP.ReadHeaderTimeout == 0 {
		cfg.HTTP.ReadHeaderTimeout = 5 * time.Second
	}
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = 65536 // 64KB
	}
	if cfg.BufferSeconds == 0 {
		cfg.BufferSeconds = 5
	}

	// env overrides
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("STREAM_SERVICE_GRPC"); v != "" {
		cfg.StreamServiceGRPC = v
	}
	if v := os.Getenv("CONTENT_SERVICE_URL"); v != "" {
		cfg.ContentServiceURL = v
	}
	if v := os.Getenv("CHUNK_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.ChunkSize = n
		}
	}
	if v := os.Getenv("BUFFER_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BufferSeconds = n
		}
	}

	return &cfg, nil
}
