package handlers

import (
	"database/sql"
	"net/http"

	"radio/user-service/infra"
)

const deleteSessionByJTI = `
	DELETE FROM sessions
	WHERE jti = $1`

func LogoutHandler(db *sql.DB) func(http.ResponseWriter, *http.Request, infra.AuthContext) {
	return func(w http.ResponseWriter, r *http.Request, actx infra.AuthContext) {
		if _, err := db.ExecContext(r.Context(), deleteSessionByJTI, actx.JTI); err != nil {
			infra.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
