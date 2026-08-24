package http

import (
	"database/sql"

	nethttp "net/http"

	httpSwagger "github.com/swaggo/http-swagger"

	health "radio/content-service/internal/api/health"
	"radio/content-service/internal/api/http/handlers"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// NewHandler builds the internal streaming API used by other microservices and
// the streaming gateway. It only serves song audio in HTTP byte-range batches;
// no user-facing JSON lives here. When swaggerEnabled is true the Swagger UI is
// served under swaggerPath and its OpenAPI document is served from disk
// (swaggerSpec), never embedded.
func NewHandler(svc *application.Services, files interfaces.FileOpener, db *sql.DB, swaggerEnabled bool, swaggerPath, swaggerSpec string) nethttp.Handler {
	mux := nethttp.NewServeMux()

	if swaggerEnabled {
		specURL := swaggerPath + "/swagger.json"
		mux.HandleFunc("GET "+specURL, func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.Header().Set("Content-Type", "application/json")
			nethttp.ServeFile(w, r, swaggerSpec)
		})
		mux.Handle(swaggerPath+"/", httpSwagger.Handler(func(c *httpSwagger.Config) {
			c.URL = specURL
		}))
	}

	mux.HandleFunc("GET /healthz", health.Check(db))
	mux.HandleFunc("GET /songs/{id}/audio", handlers.SongAudio(svc, files))
	return mux
}
