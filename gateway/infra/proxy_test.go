package infra

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsAudioPath(t *testing.T) {
	cases := map[string]bool{
		"/songs/abc/audio":           true,
		"/songs/a/b/audio":           true,
		"/songs/abc":                 false,
		"/songs/abc/audio/extra":     false,
		"/songs/":                    false,
		"/images/x/file":             false,
		"/songs/audio":               true, // degenerate but matches the prefix/suffix rule
		"/api/content/songs/1/audio": false,
	}
	for path, want := range cases {
		got := IsAudioPath(path)
		if got != want {
			t.Errorf("IsAudioPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestNewProxy_StripsPrefix(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	p := NewProxy(upstream.URL, "/api/auth")
	r := httptest.NewRequest(http.MethodGet, "http://gw/api/auth/login", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, r)

	if got := <-seen; got != "/login" {
		t.Fatalf("upstream path = %q, want /login", got)
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rr.Code)
	}
}

func TestNewProxy_ForwardsQuery(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.URL.RawQuery
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	p := NewProxy(upstream.URL, "/api/content")
	r := httptest.NewRequest(http.MethodGet, "http://gw/api/content/songs?q=deep+house", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, r)

	if got := <-seen; got != "q=deep+house" {
		t.Fatalf("query = %q", got)
	}
}

func TestNewProxy_UnreachableUpstreamReturns502(t *testing.T) {
	// A well-formed but unreachable upstream must yield a proxy error (502),
	// not hang or crash.
	p := NewProxy("http://127.0.0.1:1", "/x")
	r := httptest.NewRequest(http.MethodGet, "http://gw/x/y", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, r)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("code = %d, want 502", rr.Code)
	}
}

func TestSenderProxy_ConvertsToHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stream/skip" {
			t.Errorf("upstream path = %q, want /stream/skip", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := NewSenderProxy(upstream.URL)
	r := httptest.NewRequest(http.MethodGet, "http://gw/api/streams/skip", nil)
	rr := httptest.NewRecorder()
	p.ServeHTTP(rr, r)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d", rr.Code)
	}
}
