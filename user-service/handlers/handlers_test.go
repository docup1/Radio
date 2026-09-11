package handlers

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"radio/user-service/infra"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type handlersHarness struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
	cfg  *infra.Config
}

func newHarness(t *testing.T) *handlersHarness {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &handlersHarness{
		db:   db,
		mock: mock,
		cfg: &infra.Config{
			JWTSecret:  []byte("test-jwt-secret"),
			Auth:       infra.AuthConfig{TokenTTL: time.Hour},
			Validation: infra.ValidationConfig{UsernameMinLength: 3, UsernameMaxLength: 32, PasswordMinLength: 8, PasswordMaxLength: 72},
			Bcrypt:     infra.BcryptConfig{Cost: 4, MaxConcurrent: 4},
		},
	}
}

func (h *handlersHarness) do(t *testing.T, hf http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	hf(rec, req)
	return rec
}

func TestRegister_Success(t *testing.T) {
	h := newHarness(t)
	userID := uuid.NewString()
	h.mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(sqlmock.AnyArg(), "alice", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
	h.mock.ExpectExec(`INSERT INTO sessions`).
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"alice","password":"password1"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), userID) || !strings.Contains(rec.Body.String(), `"alice"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRegister_BadUsername(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"$$$","password":"password1"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRegister_ShortUsername(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"ab","password":"password1"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"alice","password":"123"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register", `{`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRegister_UsernameTaken(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectQuery(`INSERT INTO users`).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"alice","password":"password1"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestRegister_DBError(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectQuery(`INSERT INTO users`).
		WillReturnError(sql.ErrConnDone)

	rec := h.do(t, RegisterHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/register",
		`{"username":"alice","password":"password1"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	h := newHarness(t)
	userID := uuid.NewString()
	hash, _ := infra.NewHasher(4).Generate(t.Context(), "password1", 4)

	h.mock.ExpectQuery(`SELECT id, password`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).AddRow(userID, string(hash)))
	h.mock.ExpectExec(`INSERT INTO sessions`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rec := h.do(t, LoginHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/login",
		`{"username":"alice","password":"password1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newHarness(t)
	hash, _ := infra.NewHasher(4).Generate(t.Context(), "password1", 4)

	h.mock.ExpectQuery(`SELECT id, password`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).AddRow(uuid.NewString(), string(hash)))

	rec := h.do(t, LoginHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/login",
		`{"username":"alice","password":"wrong"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectQuery(`SELECT id, password`).
		WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	rec := h.do(t, LoginHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/login",
		`{"username":"ghost","password":"password1"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLogin_EmptyFields(t *testing.T) {
	h := newHarness(t)
	rec := h.do(t, LoginHandler(h.db, h.cfg, infra.NewHasher(4)), http.MethodPost, "/login",
		`{"username":"","password":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestMe_Success(t *testing.T) {
	h := newHarness(t)
	userID := uuid.NewString()
	h.mock.ExpectQuery(`SELECT id, username`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(userID, "alice"))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	MeHandler(h.db)(rec, req, infra.AuthContext{UserID: userID})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"alice"`) {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestMe_UnknownUser(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectQuery(`SELECT id, username`).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	MeHandler(h.db)(rec, httptest.NewRequest(http.MethodGet, "/me", nil), infra.AuthContext{UserID: "x"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestLogout_Success(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectExec(`DELETE FROM sessions`).
		WithArgs("jti-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	LogoutHandler(h.db)(rec, httptest.NewRequest(http.MethodPost, "/logout", nil), infra.AuthContext{JTI: "jti-1"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestDeleteMe_Success(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectExec(`DELETE FROM sessions`).WithArgs("user-1").WillReturnResult(sqlmock.NewResult(0, 1))
	h.mock.ExpectExec(`DELETE FROM users`).WithArgs("user-1").WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	DeleteMeHandler(h.db)(rec, httptest.NewRequest(http.MethodDelete, "/me", nil), infra.AuthContext{UserID: "user-1"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestHealth_Up(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectPing()

	rec := h.do(t, HealthHandler(h.db), http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestHealth_DBDown(t *testing.T) {
	h := newHarness(t)
	h.mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	rec := h.do(t, HealthHandler(h.db), http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestPassword_Success(t *testing.T) {
	h := newHarness(t)
	userID := uuid.NewString()
	oldHash, _ := infra.NewHasher(4).Generate(t.Context(), "oldpass123", 4)

	h.mock.ExpectQuery(`SELECT password`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(string(oldHash)))
	h.mock.ExpectExec(`UPDATE users`).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPut, "/password", bytes.NewReader([]byte(
		`{"old_password":"oldpass123","new_password":"newpass123"}`)))
	rec := httptest.NewRecorder()
	PasswordHandler(h.db, h.cfg, infra.NewHasher(4))(rec, req, infra.AuthContext{UserID: userID})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestPassword_WrongOldPassword(t *testing.T) {
	h := newHarness(t)
	userID := uuid.NewString()
	oldHash, _ := infra.NewHasher(4).Generate(t.Context(), "oldpass123", 4)

	h.mock.ExpectQuery(`SELECT password`).
		WillReturnRows(sqlmock.NewRows([]string{"password"}).AddRow(string(oldHash)))

	req := httptest.NewRequest(http.MethodPut, "/password", bytes.NewReader([]byte(
		`{"old_password":"nope","new_password":"newpass123"}`)))
	rec := httptest.NewRecorder()
	PasswordHandler(h.db, h.cfg, infra.NewHasher(4))(rec, req, infra.AuthContext{UserID: userID})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestPassword_WeakNewPassword(t *testing.T) {
	h := newHarness(t)
	req := httptest.NewRequest(http.MethodPut, "/password", bytes.NewReader([]byte(
		`{"old_password":"oldpass123","new_password":"123"}`)))
	rec := httptest.NewRecorder()
	PasswordHandler(h.db, h.cfg, infra.NewHasher(4))(rec, req, infra.AuthContext{UserID: "u"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d", rec.Code)
	}
}
