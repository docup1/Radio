package health

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// Check returns an HTTP handler that reports service health. It currently
// verifies database connectivity, returning 200 {"status":"ok"} when healthy
// and 503 {"status":"unhealthy"} otherwise.
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Failure	503	{object}	map[string]string
//	@Router		/healthz [get]
func Check(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
