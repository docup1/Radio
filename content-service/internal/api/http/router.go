package http

import (
	nethttp "net/http"

	"radio/content-service/internal/api/http/handlers"
	"radio/content-service/internal/application"
	"radio/content-service/internal/domain/interfaces"
)

// NewHandler builds the internal streaming API used by other microservices and
// the streaming gateway. It only serves song audio in HTTP byte-range batches;
// no user-facing JSON lives here.
func NewHandler(svc *application.Services, files interfaces.FileOpener) nethttp.Handler {
	mux := nethttp.NewServeMux()
	mux.HandleFunc("GET /songs/{id}/audio", handlers.SongAudio(svc, files))
	return mux
}
