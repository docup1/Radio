package infra

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	gatewayGoodToken = "good-token-123"
	gatewayUserID    = "7c7c0000-0000-0000-0000-000000000001"
)

func fakeUserService(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me" {
			t.Errorf("path = %q, want /me", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+gatewayGoodToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + gatewayUserID + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testCookieCfg() CookieConfig {
	return CookieConfig{Name: "access_token", Path: "/", MaxAge: 3600, SecureInProd: true}
}

func TestExtractToken_CookieFirst(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	r := httptest.NewRequest(http.MethodGet, "http://gw/", nil)
	r.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})
	r.Header.Set("Authorization", "Bearer header-token")

	if got := s.ExtractToken(r); got != "cookie-token" {
		t.Fatalf("token = %q, want cookie-token", got)
	}
}

func TestExtractToken_BearerFallback(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	r := httptest.NewRequest(http.MethodGet, "http://gw/", nil)
	r.Header.Set("Authorization", "Bearer header-token")
	if got := s.ExtractToken(r); got != "header-token" {
		t.Fatalf("token = %q", got)
	}
}

func TestExtractToken_Empty(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	r := httptest.NewRequest(http.MethodGet, "http://gw/", nil)
	r.AddCookie(&http.Cookie{Name: "other", Value: "x"})
	if got := s.ExtractToken(r); got != "" {
		t.Fatalf("token = %q, want empty", got)
	}
}

func TestValidate_Ok(t *testing.T) {
	s := NewAuthService(&http.Client{Timeout: 2 * time.Second}, fakeUserService(t).URL, testCookieCfg(), "dev")
	id, err := s.Validate(context.Background(), gatewayGoodToken)
	if err != nil {
		t.Fatal(err)
	}
	if id != gatewayUserID {
		t.Fatalf("id = %q", id)
	}
}

func TestValidate_RejectsBadToken(t *testing.T) {
	s := NewAuthService(&http.Client{Timeout: 2 * time.Second}, fakeUserService(t).URL, testCookieCfg(), "dev")
	if _, err := s.Validate(context.Background(), "nope"); err == nil {
		t.Fatal("want error for bad token")
	}
}

func TestSetCookie_Attrs(t *testing.T) {
	dev := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	res := &http.Response{Header: http.Header{}}
	dev.SetCookie(res, "abc", 100)
	sc := res.Header.Get("Set-Cookie")
	for _, want := range []string{"access_token=abc", "Path=/", "HttpOnly", "SameSite=Lax", "Max-Age=100"} {
		if !strings.Contains(sc, want) {
			t.Errorf("Set-Cookie %q missing %q", sc, want)
		}
	}
	if strings.Contains(sc, "Secure") {
		t.Errorf("dev cookie must not be Secure: %q", sc)
	}

	prod := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "prod")
	res2 := &http.Response{Header: http.Header{}}
	prod.SetCookie(res2, "abc", 100)
	if !strings.Contains(res2.Header.Get("Set-Cookie"), "Secure") {
		t.Errorf("prod cookie must be Secure: %q", res2.Header.Get("Set-Cookie"))
	}
}

func TestSetCookie_NoSecureWhenFlagOff(t *testing.T) {
	cfg := testCookieCfg()
	cfg.SecureInProd = false
	prod := NewAuthService(&http.Client{}, "http://u", cfg, "prod")
	res := &http.Response{Header: http.Header{}}
	prod.SetCookie(res, "abc", 100)
	if strings.Contains(res.Header.Get("Set-Cookie"), "Secure") {
		t.Errorf("cookie must not be Secure when SecureInProd=false: %q", res.Header.Get("Set-Cookie"))
	}
}

func TestClearCookie(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	res := &http.Response{Header: http.Header{}}
	s.ClearCookie(res)
	sc := res.Header.Get("Set-Cookie")
	for _, want := range []string{"access_token=", "Max-Age=0", "HttpOnly"} {
		if !strings.Contains(sc, want) {
			t.Errorf("cleared cookie %q missing %q", sc, want)
		}
	}
}

func TestSetCookieFromBody(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	body := `{"id":"u1","token":"tok-42","username":"alice"}`
	res := &http.Response{
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	if err := s.SetCookieFromBody(res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Header.Get("Set-Cookie"), "tok-42") {
		t.Fatalf("cookie missing token: %q", res.Header.Get("Set-Cookie"))
	}
	got, _ := io.ReadAll(res.Body)
	if string(got) != body {
		t.Fatalf("body not passed through: %s", got)
	}
}

func TestSetCookieFromBody_NoToken(t *testing.T) {
	s := NewAuthService(&http.Client{}, "http://u", testCookieCfg(), "dev")
	res := &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader(`{"error":"bad creds"}`)),
	}
	if err := s.SetCookieFromBody(res); err != nil {
		t.Fatal(err)
	}
	if got := res.Header.Get("Set-Cookie"); got != "" {
		t.Fatalf("unexpected cookie: %q", got)
	}
}

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusUnauthorized, "missing authentication")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != `{"error":"missing authentication"}` {
		t.Fatalf("body = %q", rr.Body.String())
	}
	if ctype := rr.Header().Get("Content-Type"); !strings.Contains(ctype, "application/json") {
		t.Fatalf("content-type = %q", ctype)
	}
}
