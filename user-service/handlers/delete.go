package handlers

import (
	"database/sql"
	"net/http"

	"radio/user-service/infra"
)

const deleteSessionsByUser = `DELETE FROM sessions WHERE user_id = $1`
const deleteUserByID = `DELETE FROM users WHERE id = $1`

// DeleteMeHandler deletes the authenticated user and all of their sessions.
//
//	@Summary	Delete account
//	@Tags		users
//	@Param		Authorization	header	string	true	"Bearer JWT"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/me [delete]
func DeleteMeHandler(db *sql.DB) func(http.ResponseWriter, *http.Request, infra.AuthContext) {
	return func(w http.ResponseWriter, r *http.Request, actx infra.AuthContext) {
		if _, err := db.ExecContext(r.Context(), deleteSessionsByUser, actx.UserID); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if _, err := db.ExecContext(r.Context(), deleteUserByID, actx.UserID); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
