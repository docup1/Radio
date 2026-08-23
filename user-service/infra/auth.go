package infra

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type AuthContext struct {
	UserID string
	JTI    string
}

const sessionExists = `
	SELECT EXISTS(
		SELECT 1
		FROM sessions
		WHERE jti = $1 AND expires_at > now()
	)`

func RequireAuth(db *sql.DB, cfg *Config, next func(http.ResponseWriter, *http.Request, AuthContext)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || tokenString == "" {
			WriteError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		parsed, err := jwt.Parse(tokenString, func(*jwt.Token) (any, error) {
			return cfg.JWTSecret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !parsed.Valid {
			WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		sub, _ := claims["sub"].(string)
		jti, _ := claims["jti"].(string)
		if sub == "" || jti == "" {
			WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		var alive bool
		if err := db.QueryRowContext(r.Context(), sessionExists, jti).Scan(&alive); err != nil {
			WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !alive {
			WriteError(w, http.StatusUnauthorized, "session expired or revoked")
			return
		}

		next(w, r, AuthContext{UserID: sub, JTI: jti})
	}
}
