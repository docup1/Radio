package infra

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func testConfigAndDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *Config) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	cfg := &Config{
		HTTP:       HTTPConfig{Addr: ":9000"},
		JWTSecret:  []byte("test-jwt-secret"),
		Auth:       AuthConfig{TokenTTL: time.Hour},
		Validation: ValidationConfig{UsernameMinLength: 3, UsernameMaxLength: 32, PasswordMinLength: 8, PasswordMaxLength: 72},
		Bcrypt:     BcryptConfig{Cost: 4, MaxConcurrent: 2},
	}
	return db, mock, cfg
}

func TestIssueToken_InsertsSessionAndSigns(t *testing.T) {
	db, mock, cfg := testConfigAndDB(t)
	userID, username := uuid.NewString(), "alice"

	mock.ExpectExec("INSERT INTO sessions").
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	signed, err := IssueToken(db, cfg, userID, username)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := jwt.Parse(signed, func(*jwt.Token) (any, error) { return cfg.JWTSecret, nil })
	if err != nil || !parsed.Valid {
		t.Fatalf("token invalid: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != userID || claims["username"] != username {
		t.Fatalf("claims = %v", claims)
	}
	if _, ok := claims["jti"].(string); !ok {
		t.Fatalf("jti missing: %v", claims)
	}
	if exp := claims["exp"].(float64); exp < float64(time.Now().Add(time.Hour-time.Minute).Unix()) {
		t.Fatalf("exp too small: %v", exp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueToken_DBFailure(t *testing.T) {
	db, mock, cfg := testConfigAndDB(t)
	mock.ExpectExec("INSERT INTO sessions").WillReturnError(sql.ErrTxDone)

	_, err := IssueToken(db, cfg, "u", "n")
	if err == nil {
		t.Fatal("want error on db failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequireAuth_NoHeader(t *testing.T) {
	db, _, cfg := testConfigAndDB(t)
	h := RequireAuth(db, cfg, func(w http.ResponseWriter, r *http.Request, a AuthContext) {
		t.Fatal("next must not be called")
	})

	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	db, _, cfg := testConfigAndDB(t)
	h := RequireAuth(db, cfg, func(w http.ResponseWriter, r *http.Request, a AuthContext) {
		t.Fatal("next must not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-token")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRequireAuth_ValidTokenHitNext(t *testing.T) {
	db, mock, cfg := testConfigAndDB(t)
	jti := uuid.NewString()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1", "jti": jti, "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString(cfg.JWTSecret)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(jti).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	var got AuthContext
	h := RequireAuth(db, cfg, func(w http.ResponseWriter, r *http.Request, a AuthContext) {
		got = a
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	req.Header.Set("X-Test", "x")

	rec := httptest.NewRecorder()
	h(rec, req)

	if got.UserID != "user-1" || got.JTI != jti {
		t.Fatalf("ctx = %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequireAuth_ExpiredSession(t *testing.T) {
	db, mock, cfg := testConfigAndDB(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1", "jti": uuid.NewString(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString(cfg.JWTSecret)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	h := RequireAuth(db, cfg, func(w http.ResponseWriter, r *http.Request, a AuthContext) {
		t.Fatal("next must not be called")
	})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d (want 401)", rec.Code)
	}
}

func TestRequireAuth_DBError(t *testing.T) {
	db, mock, cfg := testConfigAndDB(t)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user-1", "jti": uuid.NewString(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString(cfg.JWTSecret)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	h := RequireAuth(db, cfg, func(w http.ResponseWriter, r *http.Request, a AuthContext) {})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d (want 500)", rec.Code)
	}
}
