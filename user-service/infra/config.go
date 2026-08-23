package infra

import (
	"errors"
	"fmt"
	"net/url"
	"os"
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

type Config struct {
	HTTP       HTTPConfig       `yaml:"http"`
	DB         DBConfig         `yaml:"db"`
	Auth       AuthConfig       `yaml:"auth"`
	Validation ValidationConfig `yaml:"validation"`
	Bcrypt     BcryptConfig     `yaml:"bcrypt"`

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

	cfg.JWTSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(cfg.JWTSecret) == 0 {
		return nil, ErrMissingJWTSecret
	}
	cfg.DBPassword = os.Getenv("DB_PASSWORD")
	if cfg.DBPassword == "" {
		return nil, ErrMissingDBPassword
	}
	return &cfg, nil
}
