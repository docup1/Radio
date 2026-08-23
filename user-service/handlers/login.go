package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"radio/user-service/infra"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

const selectUserByUsername = `
	SELECT id, password
	FROM users
	WHERE username = $1`

const insertSession = `
	INSERT INTO sessions (jti, user_id, expires_at)
	VALUES ($1, $2, $3)`

func LoginHandler(db *sql.DB, cfg *infra.Config, hasher *infra.Hasher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			infra.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Username == "" || req.Password == "" {
			infra.WriteError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		var id, hash string
		err := db.QueryRowContext(r.Context(), selectUserByUsername, req.Username).Scan(&id, &hash)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			infra.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		case err != nil:
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if hasher.Compare(r.Context(), []byte(hash), []byte(req.Password)) != nil {
			infra.WriteError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}

		sessionID, err := uuid.NewV7()
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		expiresAt := time.Now().Add(cfg.Auth.TokenTTL)
		if _, err := db.ExecContext(r.Context(), insertSession, sessionID, id, expiresAt); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		now := time.Now()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":      id,
			"jti":      sessionID.String(),
			"username": req.Username,
			"iat":      now.Unix(),
			"exp":      expiresAt.Unix(),
		})
		signed, err := token.SignedString(cfg.JWTSecret)
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		infra.WriteJSON(w, http.StatusOK, loginResponse{Token: signed})
	}
}
