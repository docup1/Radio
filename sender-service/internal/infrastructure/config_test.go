package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("env: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Env != "test" {
		t.Fatalf("Env = %q, want test", cfg.Env)
	}
	if cfg.HTTP.Addr != ":8081" {
		t.Fatalf("HTTP.Addr = %q, want :8081", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("ReadHeaderTimeout = %v, want 5s", cfg.HTTP.ReadHeaderTimeout)
	}
	if cfg.ChunkSize != 65536 {
		t.Fatalf("ChunkSize = %d, want 65536", cfg.ChunkSize)
	}
	if cfg.Bitrate != 128000 {
		t.Fatalf("Bitrate = %d, want 128000", cfg.Bitrate)
	}
	if cfg.BufferSeconds != 5 || cfg.PrefetchCount != 8 || cfg.NextSongPrefetch != 1 {
		t.Fatalf("buffer/prefetch defaults wrong: %+v", cfg)
	}
	if cfg.Redis != "redis:6379" {
		t.Fatalf("Redis = %q, want redis:6379", cfg.Redis)
	}
}

func TestLoadConfigYAMLOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	body := "http:\n  addr: \":9999\"\nchunk_size: 8192\nbuffer_seconds: 3\nredis: \"localhost:7000\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.HTTP.Addr != ":9999" || cfg.ChunkSize != 8192 || cfg.BufferSeconds != 3 || cfg.Redis != "localhost:7000" {
		t.Fatalf("overrides not applied: %+v", cfg)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	if _, err := LoadConfig("/nonexistent/nope.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	t.Setenv("REDIS_ADDR", "env-redis:7000")
	t.Setenv("CHUNK_SIZE", "1024")
	t.Setenv("HTTP_ADDR", ":1111")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Redis != "env-redis:7000" || cfg.ChunkSize != 1024 || cfg.HTTP.Addr != ":1111" {
		t.Fatalf("env overrides not applied: %+v", cfg)
	}
}
