package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
	svc *application.Service
}

func NewHandler(hub *application.Hub, svc *application.Service) *Handler {
	return &Handler{hub: hub, svc: svc}
}

// WebSocket protocol (unified: one socket carries audio, control and events).
//
// Client → server (text JSON):
//
//	{"type":"start","loop":bool}   — owner starts the stream
//	{"type":"stop"}                — owner stops the stream
//	{"type":"skip"}                — owner skips to the next song
//	{"type":"ping"}                — keepalive / connectivity check
//
// Server → client (text JSON):
//
//	{"type":"state","data":{is_active,current_song_id,current_item_id,position,queue_length}}
//	{"type":"song","songId":"..."}            — song started
//	{"type":"song_ended","songId":"..."}      — song finished (auto or skip)
//	{"type":"stream_ended","message":"..."}   — queue exhausted, auto-stopped
//	{"type":"stream_stopped"}                 — playback stopped by owner/REST
//	{"type":"error","message":"..."}          — a control command was rejected
//
// Binary messages carry MPEG-frame-aligned audio buffers.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" || r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		return
	}

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

	// GetOrCreate so listeners can connect to inactive streams too — the hub
	// outlives stop so the same socket can restart without reconnecting.
	sh := h.hub.GetOrCreate(streamID, 256)
	if sh == nil {
		http.Error(w, "stream not active", http.StatusNotFound)
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
		ID:        strconv.FormatInt(listenerID, 10),
		Ch:        make(chan []byte, 1024),
		ControlCh: make(chan string, 64),
		OwnerID:   r.Header.Get("X-Owner-ID"),
		Hub:       sh,
	}

	sh.Subscribe(l)
	log.Printf("[ws] listener %s connected to stream %s (owner=%q, total: %d)",
		l.ID, streamID, l.OwnerID, len(sh.Listeners))

	// Initial state snapshot
	h.svc.SendStateTo(streamID, l)

	// Read loop: process control commands; also detects disconnect.
	go func() {
		defer func() {
			sh.Unsubscribe(l)
			conn.Close()
			log.Printf("[ws] listener %s disconnected from stream %s", l.ID, streamID)
		}()

		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[ws] read loop exit (stream=%s listener=%s): %v", streamID, l.ID, err)
				return
			}
			if msgType != websocket.TextMessage {
				continue // binary from a client is ignored
			}
			h.handleCommand(streamID, l, data)
		}
	}()

	// Write loop: multiplex binary chunks and text control messages.
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for {
			select {
			case chunk, ok := <-l.Ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.BinaryMessage, chunk); err != nil {
					log.Printf("[ws] write error (stream=%s listener=%s): %v", streamID, l.ID, err)
					return
				}
			case msg, ok := <-l.ControlCh:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					log.Printf("[ws] write error (stream=%s listener=%s): %v", streamID, l.ID, err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

type command struct {
	Type string `json:"type"`
	Loop bool   `json:"loop"`
}

func (h *Handler) handleCommand(streamID uuid.UUID, l *application.Listener, data []byte) {
	var cmd command
	if err := json.Unmarshal(data, &cmd); err != nil || cmd.Type == "" {
		return
	}

	// Only the owner (stream.ID == owner user ID) may control playback.
	isOwner := l.OwnerID != "" && l.OwnerID == streamID.String()

	switch cmd.Type {
	case "start":
		if !isOwner {
			h.reject(l, "только владелец может запустить стрим")
			return
		}
		if err := h.svc.Start(streamID, cmd.Loop); err != nil {
			h.reject(l, err.Error())
		}
	case "stop":
		if !isOwner {
			h.reject(l, "только владелец может остановить стрим")
			return
		}
		if err := h.svc.Stop(streamID); err != nil {
			h.reject(l, err.Error())
		}
	case "skip":
		if !isOwner {
			h.reject(l, "только владелец может переключить песню")
			return
		}
		if _, err := h.svc.Skip(streamID); err != nil {
			h.reject(l, err.Error())
		}
	case "ping":
		l.Hub.SendControlToOne(l, []byte(`{"type":"pong"}`))
	default:
		// ignore unknown commands
	}
}

func (h *Handler) reject(l *application.Listener, msg string) {
	data, _ := json.Marshal(map[string]string{"type": "error", "message": msg})
	l.Hub.SendControlToOne(l, data)
}