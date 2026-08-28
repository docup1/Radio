package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"radio/sender-service/internal/application"
)

type StatusHandler struct {
	hub *application.Hub
}

func NewStatusHandler(hub *application.Hub) *StatusHandler {
	return &StatusHandler{hub: hub}
}

type StatusResponse struct {
	StreamID        string  `json:"stream_id"`
	Active          bool    `json:"active"`
	SongID          string  `json:"song_id,omitempty"`
	PositionSeconds float64 `json:"position_seconds"`
	BytesSent       int64   `json:"bytes_sent"`
	FileSize        int64   `json:"file_size"`
}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	path = strings.TrimSuffix(path, "/status")
	if path == "" || path == r.URL.Path {
		http.Error(w, "missing stream id", http.StatusBadRequest)
		return
	}

	streamID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	sh := h.hub.Get(streamID)
	if sh == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StatusResponse{
			StreamID: streamID.String(),
			Active:   false,
		})
		return
	}

	sh.RLock()
	songID := sh.SongID
	bytesSent := sh.BytesSent
	fileSize := sh.FileSize
	startedAt := sh.StartedAt
	sh.RUnlock()

	resp := StatusResponse{
		StreamID:  streamID.String(),
		Active:    true,
		BytesSent: bytesSent,
		FileSize:  fileSize,
	}

	if songID != uuid.Nil {
		resp.SongID = songID.String()
		resp.PositionSeconds = time.Since(time.Unix(0, startedAt)).Seconds()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
