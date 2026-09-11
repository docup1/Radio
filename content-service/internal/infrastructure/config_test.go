package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const contentTestYAML = `---
env: dev
log_level: info
http_public:
  addr: ":8080"
http_private:
  addr: ":8081"
grpc:
  addr: ":50051"
db:
  host: localhost
  port: 5432
  user: radio
  name: radio
  sslmode: disable
storage:
  chunk_dir: /data/chunks
  final_dir: /data/media
  max_chunk_size: 4194304
  max_file_size: 52428800
swagger:
  enabled: true
  path: /docs
  spec_file: docs/api.json
`

func writeContentConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_RequiresDBPassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")
	if _, err := LoadConfig(writeContentConfig(t, contentTestYAML)); err != ErrMissingDBPassword {
		t.Fatalf("err = %v, want ErrMissingDBPassword", err)
	}
}

func TestLoadConfig_ReadsAndOverrides(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	cfg, err := LoadConfig(writeContentConfig(t, contentTestYAML))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DBPassword != "secret" || cfg.DB.Host != "localhost" || cfg.DB.Port != 5432 {
		t.Fatalf("db = %+v", cfg.DB)
	}
	if cfg.Storage.MaxChunkSize != 4194304 || cfg.Storage.MaxFileSize != 52428800 {
		t.Fatalf("storage = %+v", cfg.Storage)
	}
	if cfg.Swagger.Path != "/docs" {
		t.Fatalf("swagger = %+v", cfg.Swagger)
	}

	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6432")
	t.Setenv("HTTP_PUBLIC_ADDR", ":9090")
	t.Setenv("STORAGE_CHUNK_DIR", "/tmp/chunks")
	cfg, err = LoadConfig(writeContentConfig(t, contentTestYAML))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "db.internal" || cfg.DB.Port != 6432 {
		t.Fatalf("env db = %+v", cfg.DB)
	}
	if cfg.HTTPPublic.Addr != ":9090" || cfg.Storage.ChunkDir != "/tmp/chunks" {
		t.Fatalf("env overrides not applied: %+v / %+v", cfg.HTTPPublic, cfg.Storage)
	}
}

func TestLoadConfig_SwaggerDefaults(t *testing.T) {
	t.Setenv("DB_PASSWORD", "x")
	minimal := "---\ndb:\n  host: h\n  port: 5432\n"
	cfg, err := LoadConfig(writeContentConfig(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Swagger.Path != "/swagger" || cfg.Swagger.SpecFile != "docs/swagger.json" {
		t.Fatalf("swagger defaults = %+v", cfg.Swagger)
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
