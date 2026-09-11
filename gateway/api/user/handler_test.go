package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"radio/gateway/infra"
)

type userHarness struct {
	h *Handler
}

type record struct {
	path          string
	authorization string
	ownerID       string
}

func newUserHarness(t *testing.T) *userHarness {
	t.Helper()
	const token = "token-abc"

	users := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/register", "/login":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"u1","token":"` + token + `"}`))
		case "/me":
			if r.Header.Get("Authorization") != "Bearer "+token {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"u1"}`))
		case "/logout":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(users.Close)

	auth := infra.NewAuthService(&http.Client{}, users.URL,
		infra.CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600}, "dev")
	proxy := infra.NewProxy(users.URL, "/api/auth")

	return &userHarness{h: New(proxy, auth)}
}

func TestAuthProxy_ProxiesWithoutToken(t *testing.T) {
	h := newUserHarness(t)
	seen := make(chan record, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- record{path: r.URL.Path, authorization: r.Header.Get("Authorization"), ownerID: r.Header.Get("X-Owner-ID")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	h.h = New(infra.NewProxy(upstream.URL, "/api/auth"), infra.NewAuthService(&http.Client{}, upstream.URL,
		infra.CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600}, "dev"))

	req := httptest.NewRequest(http.MethodPost, "http://gw/api/auth/login", nil)
	req.Header.Set("X-Owner-ID", "spoofed")
	rr := httptest.NewRecorder()
	h.h.authProxy(false)(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rr.Code)
	}
	got := <-seen
	if got.path != "/login" {
		t.Fatalf("path = %q", got.path)
	}
	if got.ownerID != "" {
		t.Fatalf("X-Owner-ID not stripped: %q", got.ownerID)
	}
	if got.authorization != "" {
		t.Fatalf("unexpected auth header: %q", got.authorization)
	}
}

func TestAuthProxy_RequiresToken(t *testing.T) {
	h := newUserHarness(t)
	rr := httptest.NewRecorder()
	h.h.authProxy(true)(rr, httptest.NewRequest(http.MethodGet, "http://gw/api/auth/me", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestAuthProxy_ForwardsCookieToken(t *testing.T) {
	h := newUserHarness(t)
	seen := make(chan record, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- record{authorization: r.Header.Get("Authorization")}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"u1"}`))
	}))
	defer upstream.Close()
	h.h = New(infra.NewProxy(upstream.URL, "/api/auth"), infra.NewAuthService(&http.Client{}, upstream.URL,
		infra.CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600}, "dev"))

	req := httptest.NewRequest(http.MethodGet, "http://gw/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	rr := httptest.NewRecorder()
	h.h.authProxy(true)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d", rr.Code)
	}
	if got := <-seen; got.authorization != "Bearer cookie-token" {
		t.Fatalf("auth = %q", got.authorization)
	}
}

func TestAuthProxy_ForwardsBearerToken(t *testing.T) {
	h := newUserHarness(t)
	seen := make(chan record, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- record{authorization: r.Header.Get("Authorization")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	h.h = New(infra.NewProxy(upstream.URL, "/api/auth"), infra.NewAuthService(&http.Client{}, upstream.URL,
		infra.CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600}, "dev"))

	req := httptest.NewRequest(http.MethodGet, "http://gw/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer bearer-token")
	rr := httptest.NewRecorder()
	h.h.authProxy(true)(rr, req)

	if got := <-seen; got.authorization != "Bearer bearer-token" {
		t.Fatalf("auth = %q", got.authorization)
	}
}
