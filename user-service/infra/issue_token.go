package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const insertSession = `
	INSERT INTO sessions (jti, user_id, expires_at)
	VALUES ($1, $2, $3)`

// IssueToken creates a new session row and returns a signed JWT carrying the
// user's id (sub) and the session id (jti). The session must exist for the token
// to be accepted by RequireAuth, so the gateway can validate tokens indirectly
// by calling the user-service /me endpoint (which re-checks the session).
func IssueToken(db *sql.DB, cfg *Config, userID, username string) (string, error) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(cfg.Auth.TokenTTL)
	if _, err := db.ExecContext(context.Background(), insertSession, sessionID, userID, expiresAt); err != nil {
		return "", err
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"jti":      sessionID.String(),
		"username": username,
		"iat":      now.Unix(),
		"exp":      expiresAt.Unix(),
	})
	return token.SignedString(cfg.JWTSecret)
}
