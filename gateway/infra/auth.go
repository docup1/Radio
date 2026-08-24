package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// AuthService provides token extraction, validation (delegated to user-service /me),
// and HttpOnly cookie management. The gateway holds no JWT secret.
type AuthService struct {
	httpClient  *http.Client
	userService string
	cookie      CookieConfig
	env         string
}

// NewAuthService builds an AuthService.
func NewAuthService(httpClient *http.Client, userService string, cookie CookieConfig, env string) *AuthService {
	return &AuthService{httpClient: httpClient, userService: userService, cookie: cookie, env: env}
}

// ExtractToken returns the token from the HttpOnly cookie first, then from an
// Authorization: Bearer header. The gateway validates whichever it finds via the
// user-service, so it never trusts a raw X-Owner-ID.
func (s *AuthService) ExtractToken(r *http.Request) string {
	if c, err := r.Cookie(s.cookie.Name); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

// Validate asks the user-service to validate the token (which re-checks the DB
// session) and returns the authenticated user id.
func (s *AuthService) Validate(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.userService+"/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("unauthorized")
	}

	var m struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	if m.ID == "" {
		return "", errors.New("unauthorized")
	}
	return m.ID, nil
}

// SetCookie writes the auth cookie onto an upstream response.
func (s *AuthService) SetCookie(res *http.Response, value string, maxAge int) {
	c := &http.Cookie{
		Name:     s.cookie.Name,
		Value:    value,
		Path:     s.cookie.Path,
		HttpOnly: true,
		Secure:   s.env == "prod" && s.cookie.SecureInProd,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
	res.Header.Set("Set-Cookie", c.String())
}

// ClearCookie expires the auth cookie (used on logout).
func (s *AuthService) ClearCookie(res *http.Response) {
	s.SetCookie(res, "", -1)
}

// SetCookieFromBody reads the upstream response body; if it carries a "token" field,
// sets it as the HttpOnly cookie. The body is passed through unchanged so
// non-browser clients can still read the token.
func (s *AuthService) SetCookieFromBody(res *http.Response) error {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body.Close()

	var m struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(body, &m)
	if m.Token != "" {
		s.SetCookie(res, m.Token, s.cookie.MaxAge)
	}

	res.Body = io.NopCloser(bytes.NewReader(body))
	return nil
}

// WriteError writes the standard JSON error envelope.
func WriteError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
