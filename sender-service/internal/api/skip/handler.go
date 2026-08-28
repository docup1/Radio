package skip

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"radio/sender-service/internal/application"
)

type Handler struct {
	svc *application.Service
}

func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	path = strings.TrimSuffix(path, "/skip")
	if path == "" || path == r.URL.Path {
		http.Error(w, "missing stream id", http.StatusBadRequest)
		return
	}

	streamID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	songID, err := h.svc.Skip(streamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if songID == nil {
		http.Error(w, "stream ended", http.StatusGone)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"song_id": songID.String()})
	log.Printf("[skip] stream=%s song=%s", streamID, *songID)
}
