package handlers

import (
	"database/sql"
	"net/http"

	"radio/user-service/infra"
)

const deleteSessionByJTI = `
	DELETE FROM sessions
	WHERE jti = $1`

// LogoutHandler revokes the current session.
//
//	@Summary	Logout (revoke session)
//	@Tags		users
//	@Param		Authorization	header	string	true	"Bearer JWT"
//	@Success	204
//	@Failure	401	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/logout [post]
func LogoutHandler(db *sql.DB) func(http.ResponseWriter, *http.Request, infra.AuthContext) {
	return func(w http.ResponseWriter, r *http.Request, actx infra.AuthContext) {
		if _, err := db.ExecContext(r.Context(), deleteSessionByJTI, actx.JTI); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
