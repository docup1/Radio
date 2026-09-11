package content

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"radio/gateway/infra"
)

func TestPathHelpers(t *testing.T) {
	cases := []struct {
		path   string
		meta   bool
		image  bool
		public bool
	}{
		{"/songs/abc123", true, false, true},
		{"/songs/abc123/audio", false, false, false},
		{"/songs/abc123/extra", false, false, false},
		{"/songs/", false, false, false},
		{"/songs/", false, false, false},
		{"/images/uuid/file", false, true, true},
		{"/images/uuid", false, false, false},
		{"/images/uuid/file/x", false, false, false},
		{"/other/thing", false, false, false},
	}
	for _, c := range cases {
		if got := isSongMetaPath(c.path); got != c.meta {
			t.Errorf("isSongMetaPath(%q) = %v, want %v", c.path, got, c.meta)
		}
		if got := isImageFilePath(c.path); got != c.image {
			t.Errorf("isImageFilePath(%q) = %v, want %v", c.path, got, c.image)
		}
		if got := publicReadPath(c.path); got != c.public {
			t.Errorf("publicReadPath(%q) = %v, want %v", c.path, got, c.public)
		}
	}
}

type contentCaptured struct {
	path       string
	query      string
	ownerID    string
	authHeader string
}

type contentHarness struct {
	h         *Handler
	rest      chan *contentCaptured
	stream    chan *contentCaptured
	goodToken string
	badToken  string
}

func newContentHarness(t *testing.T) *contentHarness {
	t.Helper()
	const goodToken = "good-token-xyz"

	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+goodToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1"}`))
	}))
	t.Cleanup(userSrv.Close)

	rest := make(chan *contentCaptured, 1)
	restSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest <- &contentCaptured{
			path: r.URL.Path, query: r.URL.RawQuery,
			ownerID: r.Header.Get("X-Owner-ID"), authHeader: r.Header.Get("Authorization"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"rest":true}`))
	}))
	t.Cleanup(restSrv.Close)

	stream := make(chan *contentCaptured, 1)
	streamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream <- &contentCaptured{
			path: r.URL.Path, query: r.URL.RawQuery,
			ownerID: r.Header.Get("X-Owner-ID"), authHeader: r.Header.Get("Authorization"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stream":true}`))
	}))
	t.Cleanup(streamSrv.Close)

	auth := infra.NewAuthService(&http.Client{}, userSrv.URL,
		infra.CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600}, "dev")

	return &contentHarness{
		h:         New(infra.NewProxy(restSrv.URL, "/api/content"), infra.NewProxy(streamSrv.URL, "/api/content"), auth),
		rest:      rest,
		stream:    stream,
		goodToken: goodToken,
		badToken:  "bad-token",
	}
}

func (h *contentHarness) do(t *testing.T, method, path string, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://gw"+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.h.content(rr, req)
	return rr
}

func TestContent_AnonymousPublicMeta(t *testing.T) {
	h := newContentHarness(t)
	rr := h.do(t, http.MethodGet, "/api/content/songs/abc123", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d body = %s", rr.Code, rr.Body.String())
	}
	got := <-h.rest
	if got.path != "/songs/abc123" || got.ownerID != "" {
		t.Fatalf("captured = %+v; want no owner id", got)
	}
}

func TestContent_AnonymousPublicImage(t *testing.T) {
	h := newContentHarness(t)
	rr := h.do(t, http.MethodGet, "/api/content/images/uuid/file", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d", rr.Code)
	}
	got := <-h.rest
	if got.path != "/images/uuid/file" {
		t.Fatalf("path = %q", got.path)
	}
}

func TestContent_AnonymousMetaWithValidTokenSetsOwner(t *testing.T) {
	h := newContentHarness(t)
	rr := h.do(t, http.MethodGet, "/api/content/songs/abc123", h.goodToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d", rr.Code)
	}
	got := <-h.rest
	if got.ownerID != "user-1" {
		t.Fatalf("owner = %q, want user-1", got.ownerID)
	}
}

func TestContent_AudioRoutesToStreamProxy(t *testing.T) {
	h := newContentHarness(t)
	req := httptest.NewRequest(http.MethodGet, "http://gw/api/content/songs/abc123/audio", nil)
	req.Header.Set("Authorization", "Bearer "+h.goodToken)
	rr := httptest.NewRecorder()
	h.h.content(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d body = %s", rr.Code, rr.Body.String())
	}
	got := <-h.stream
	if got.path != "/songs/abc123/audio" {
		t.Fatalf("path = %q", got.path)
	}
	if got.ownerID != "user-1" {
		t.Fatalf("owner = %q, want user-1", got.ownerID)
	}
	if !strings.Contains(got.query, "user_id=user-1") {
		t.Fatalf("query = %q, want user_id", got.query)
	}
}

func TestContent_AuthRequired(t *testing.T) {
	h := newContentHarness(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/content/songs"},
		{http.MethodGet, "/api/content/songs/abc123/audio"},
		{http.MethodPut, "/api/content/songs/abc123"},
		{http.MethodDelete, "/api/content/songs/abc123"},
	} {
		rr := h.do(t, tc.method, tc.path, "")
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: code = %d, want 401", tc.method, tc.path, rr.Code)
		}
	}
}

func TestContent_RejectsInvalidToken(t *testing.T) {
	h := newContentHarness(t)
	// audio without valid token -> 401
	rr := h.do(t, http.MethodGet, "/api/content/songs/abc123/audio", h.badToken)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}

	// public meta with invalid token: anonymous pass-through (no owner set)
	rr = h.do(t, http.MethodGet, "/api/content/songs/abc123", h.badToken)
	if rr.Code != http.StatusOK {
		t.Fatalf("public meta code = %d", rr.Code)
	}
	got := <-h.rest
	if got.ownerID != "" {
		t.Fatalf("owner = %q, want empty for invalid token", got.ownerID)
	}
}

func TestContent_StripsClientIdentityHeaders(t *testing.T) {
	h := newContentHarness(t)
	req := httptest.NewRequest(http.MethodGet, "http://gw/api/content/songs/abc123", nil)
	req.Header.Set("X-Owner-ID", "spoofed")
	rr := httptest.NewRecorder()
	h.h.content(rr, req)
	got := <-h.rest
	if got.ownerID != "" && got.ownerID != "spoofed" {
		t.Fatalf("owner = %q", got.ownerID)
	}
	// never a spoofed X-Owner-ID reaching the upstream without validation
	if strings.Contains(rr.Header().Get("X-Owner-ID"), "spoofed") {
		t.Fatalf("leaked spoofed header")
	}
}
