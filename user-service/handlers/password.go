package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"radio/user-service/infra"
)

type passwordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

const selectPasswordByID = `
	SELECT password
	FROM users
	WHERE id = $1`

const updateUserPassword = `
	UPDATE users
	SET password = $1
	WHERE id = $2`

// PasswordHandler changes the authenticated user's password.
//
//	@Summary	Change password
//	@Tags		users
//	@Accept		json
//	@Param		Authorization	header	string			true	"Bearer JWT"
//	@Param		request			body		passwordRequest	true	"Password change payload"
//	@Success	204
//	@Failure	400	{object}	map[string]string
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/password [put]
func PasswordHandler(db *sql.DB, cfg *infra.Config, hasher *infra.Hasher) func(http.ResponseWriter, *http.Request, infra.AuthContext) {
	return func(w http.ResponseWriter, r *http.Request, actx infra.AuthContext) {
		var req passwordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			infra.WriteError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.OldPassword == "" || req.NewPassword == "" {
			infra.WriteError(w, http.StatusBadRequest, "old_password and new_password are required")
			return
		}
		if !cfg.Validation.ValidPassword(req.NewPassword) {
			infra.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
				"password must be %d-%d characters",
				cfg.Validation.PasswordMinLength,
				cfg.Validation.PasswordMaxLength,
			))
			return
		}

		var hash string
		err := db.QueryRowContext(r.Context(), selectPasswordByID, actx.UserID).Scan(&hash)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			infra.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		case err != nil:
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if hasher.Compare(r.Context(), []byte(hash), []byte(req.OldPassword)) != nil {
			infra.WriteError(w, http.StatusBadRequest, "invalid old password")
			return
		}

		newHash, err := hasher.Generate(r.Context(), req.NewPassword, cfg.Bcrypt.Cost)
		if err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if _, err := db.ExecContext(r.Context(), updateUserPassword, newHash, actx.UserID); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
