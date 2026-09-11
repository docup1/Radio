package infra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const userTestYAML = `---
http:
  addr: ":9000"
  read_header_timeout: 3s
db:
  host: localhost
  port: 5432
  user: radio
  name: radio
  sslmode: disable
auth:
  token_ttl: 24h
  cleanup_interval: 1h
validation:
  username_min_length: 3
  username_max_length: 32
  password_min_length: 8
  password_max_length: 72
bcrypt:
  cost: 10
  max_concurrent: 4
swagger:
  enabled: true
  path: /docs
  spec_file: docs/api.json
`

func writeUserConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_RequiresSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_PASSWORD", "")
	if _, err := LoadConfig(writeUserConfig(t, userTestYAML)); err != ErrMissingJWTSecret {
		t.Fatalf("err = %v, want ErrMissingJWTSecret", err)
	}

	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("DB_PASSWORD", "")
	if _, err := LoadConfig(writeUserConfig(t, userTestYAML)); err != ErrMissingDBPassword {
		t.Fatalf("err = %v, want ErrMissingDBPassword", err)
	}
}

func TestLoadConfig_ReadsAndDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("DB_PASSWORD", "dbpw")

	cfg, err := LoadConfig(writeUserConfig(t, userTestYAML))
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.JWTSecret) != "s3cret" || cfg.DBPassword != "dbpw" {
		t.Fatalf("secrets not applied")
	}
	if cfg.Auth.TokenTTL != 24*time.Hour || cfg.Auth.CleanupInterval != time.Hour {
		t.Fatalf("auth = %+v", cfg.Auth)
	}
	if cfg.Validation.PasswordMinLength != 8 || cfg.Validation.PasswordMaxLength != 72 {
		t.Fatalf("validation = %+v", cfg.Validation)
	}
	if cfg.Bcrypt.Cost != 10 || cfg.Swagger.Path != "/docs" {
		t.Fatalf("bcrypt/swagger = %+v", cfg.Bcrypt)
	}

	t.Setenv("SWAGGER_ENABLED", "1")
	t.Setenv("SWAGGER_PATH", "/envdocs")
	cfg, err = LoadConfig(writeUserConfig(t, userTestYAML))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Swagger.Enabled || cfg.Swagger.Path != "/envdocs" {
		t.Fatalf("swagger env = %+v", cfg.Swagger)
	}
}

func TestLoadConfig_SwaggerDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("DB_PASSWORD", "p")
	cfg, err := LoadConfig(writeUserConfig(t, "---\ndb:\n  host: h\n  port: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Swagger.Path != "/swagger" || cfg.Swagger.SpecFile != "docs/swagger.json" {
		t.Fatalf("swagger defaults = %+v", cfg.Swagger)
	}
}

func TestValidPassword_Boundaries(t *testing.T) {
	v := ValidationConfig{PasswordMinLength: 8, PasswordMaxLength: 72}
	cases := map[string]bool{
		strings.Repeat("x", 7):  false,
		strings.Repeat("x", 8):  true,
		strings.Repeat("x", 72): true,
		strings.Repeat("x", 73): false,
		"":                      false,
	}
	for pw, want := range cases {
		if got := v.ValidPassword(pw); got != want {
			t.Errorf("ValidPassword(len=%d) = %v, want %v", len(pw), got, want)
		}
	}
}

func TestDBConfig_DSN(t *testing.T) {
	c := DBConfig{Host: "localhost", Port: 5432, User: "radio", Name: "radio", SSLMode: "disable"}
	dsn := c.DSN("pw")
	for _, want := range []string{"postgres://", "radio:", "pw@localhost:5432", "/radio", "sslmode=disable"} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN %q missing %q", dsn, want)
		}
	}
}
