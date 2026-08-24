package websocket

import (
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"radio/sender-service/internal/application"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var listenerCounter int64

type Handler struct {
	hub *application.Hub
}

func NewHandler(hub *application.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse stream ID from path: /stream/{id}
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	if path == "" || path == r.URL.Path {
		http.Error(w, "missing stream id", http.StatusBadRequest)
		return
	}

	streamID, err := uuid.Parse(path)
	if err != nil {
		http.Error(w, "invalid stream id", http.StatusBadRequest)
		return
	}

	// Check if stream exists
	sh := h.hub.Get(streamID)
	if sh == nil {
		http.Error(w, "stream not found", http.StatusNotFound)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	listenerID := atomic.AddInt64(&listenerCounter, 1)
	l := &application.Listener{
		ID:  string(rune(listenerID)),
		Ch:  make(chan []byte, 100),
		Hub: sh,
	}

	sh.Subscribe(l)
	log.Printf("[ws] listener %s connected to stream %s (total: %d)", l.ID, streamID, len(sh.Listeners))

	// Read loop (detect disconnect)
	go func() {
		defer func() {
			sh.Unsubscribe(l)
			conn.Close()
			log.Printf("[ws] listener %s disconnected from stream %s", l.ID, streamID)
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Write loop
	go func() {
		for chunk := range l.Ch {
			if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
				log.Printf("[ws] write error: %v", err)
				return
			}
		}
	}()
}
