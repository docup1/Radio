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

type RedisConfig struct {
	Addr     string        `yaml:"addr"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
Timeout time.Duration `yaml:"connect_timeout"`
}

type Config struct {
	Env                string        `yaml:"env"`
	LogLevel           string        `yaml:"log_level"`
	HTTP               HTTPConfig    `yaml:"http"`
	DB                 DBConfig      `yaml:"db"`
	Redis              RedisConfig   `yaml:"redis"`
	Swagger            SwaggerConfig `yaml:"swagger"`
	ContentServiceGRPC string        `yaml:"content_service_grpc"`

	DBPassword string
}

type SwaggerConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Path     string `yaml:"path"`
	SpecFile string `yaml:"spec_file"`
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

	// env overrides
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
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("SWAGGER_ENABLED"); v != "" {
		cfg.Swagger.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("CONTENT_SERVICE_GRPC"); v != "" {
		cfg.ContentServiceGRPC = v
	}

	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	if cfg.DBPassword == "" {
		return nil, ErrMissingDBPassword
	}
	return &cfg, nil
}
