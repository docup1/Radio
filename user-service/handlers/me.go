package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"radio/user-service/infra"
)

type meResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

const selectUserByID = `
	SELECT id, username
	FROM users
	WHERE id = $1`

// MeHandler returns the currently authenticated user.
//
//	@Summary	Get current user
//	@Tags		users
//	@Produce	json
//	@Param		Authorization	header	string	true	"Bearer JWT"
//	@Success	200	{object}	meResponse
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/me [get]
func MeHandler(db *sql.DB) func(http.ResponseWriter, *http.Request, infra.AuthContext) {
	return func(w http.ResponseWriter, r *http.Request, actx infra.AuthContext) {
		var resp meResponse
		err := db.QueryRowContext(r.Context(), selectUserByID, actx.UserID).Scan(&resp.ID, &resp.Username)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			infra.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		case err != nil:
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		infra.WriteJSON(w, http.StatusOK, resp)
	}
}
