package infrastructure

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type HTTPConfig struct {
	Addr              string        `yaml:"addr"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
}

type Config struct {
	Env               string     `yaml:"env"`
	LogLevel          string     `yaml:"log_level"`
	HTTP              HTTPConfig `yaml:"http"`
	Redis             string     `yaml:"redis"`
	ContentServiceURL string     `yaml:"content_service_url"`
	ChunkSize         int64      `yaml:"chunk_size"`
	Bitrate           int64      `yaml:"bitrate"`
	BufferSeconds     int        `yaml:"buffer_seconds"`
	PrefetchCount     int        `yaml:"prefetch_count"`
	NextSongPrefetch  int        `yaml:"next_song_prefetch"`
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
		cfg.ChunkSize = 65536
	}
	if cfg.Bitrate == 0 {
		cfg.Bitrate = 128000
	}
	if cfg.BufferSeconds == 0 {
		cfg.BufferSeconds = 5
	}
	if cfg.PrefetchCount == 0 {
		cfg.PrefetchCount = 8
	}
	if cfg.NextSongPrefetch == 0 {
		cfg.NextSongPrefetch = 1
	}
	if cfg.Redis == "" {
		cfg.Redis = "redis:6379"
	}

	// env overrides
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.Env = v
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis = v
	}
	if v := os.Getenv("CONTENT_SERVICE_URL"); v != "" {
		cfg.ContentServiceURL = v
	}
	if v := os.Getenv("CHUNK_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.ChunkSize = n
		}
	}
	if v := os.Getenv("BITRATE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Bitrate = n
		}
	}
	if v := os.Getenv("BUFFER_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BufferSeconds = n
		}
	}
	if v := os.Getenv("PREFETCH_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.PrefetchCount = n
		}
	}
	if v := os.Getenv("NEXT_SONG_PREFETCH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.NextSongPrefetch = n
		}
	}

	return &cfg, nil
}
