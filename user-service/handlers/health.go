package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"radio/user-service/infra"
)

// HealthHandler reports DB connectivity.
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/healthz [get]
func HealthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			infra.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		infra.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
