package infra

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupStatic(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestStaticHandler_ServesIndex(t *testing.T) {
	h := StaticHandler(setupStatic(t))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "app") {
		t.Fatalf("index: code=%d body=%q", rr.Code, rr.Body.String())
	}

	// SPA fallback for unknown route
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/streams/abc", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "app") {
		t.Fatalf("fallback: code=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestStaticHandler_ServesRealFile(t *testing.T) {
	dir := setupStatic(t)
	h := StaticHandler(dir)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "console.log(1)" {
		t.Fatalf("file: code=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestStaticHandler_RejectsTraversalAndMethods(t *testing.T) {
	dir := setupStatic(t)
	h := StaticHandler(dir)

	// Traversal-like paths are normalized (leading ".." collapse under "/"), so they
	// can never escape base; any non-existent file falls back to index.html (SPA).
	for _, p := range []string{"/../../etc/passwd", "/..%2f..%2f..%2fetc/secret", "/%2e%2e/%2e%2e/bin/sh"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
		if body := rr.Body.String(); strings.Contains(body, "root:") ||
			strings.Contains(body, "app") == false {
			t.Fatalf("path %q: body=%q, want SPA index (no system file)", p, body)
		}
	}

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("POST code = %d", rr.Code)
	}
}

func TestStaticHandler_MissingDirIndex(t *testing.T) {
	h := StaticHandler(filepath.Join(t.TempDir(), "empty"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d", rr.Code)
	}
}
