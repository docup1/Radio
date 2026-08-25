package application

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
)

// Hub manages per-stream state: audio buffer and WebSocket listeners.
type Hub struct {
	mu      sync.RWMutex
	streams map[uuid.UUID]*StreamHub
}

type StreamHub struct {
	StreamID  uuid.UUID
	Revision  int64
	SongID    uuid.UUID
	StartedAt int64 // unix nanos

	// Audio buffer: last N chunks
	Chunks [][]byte
	MaxCap int

	// WebSocket listeners
	Listeners map[*Listener]struct{}
	mu        sync.RWMutex

	cancel context.CancelFunc
}

type Listener struct {
	ID   string
	Ch   chan []byte
	Hub  *StreamHub
	Once bool // sent initial state
}

func NewHub() *Hub {
	return &Hub{
		streams: make(map[uuid.UUID]*StreamHub),
	}
}

func (h *Hub) GetOrCreate(streamID uuid.UUID, maxChunks int) *StreamHub {
	h.mu.Lock()
	defer h.mu.Unlock()

	if sh, ok := h.streams[streamID]; ok {
		return sh
	}

	sh := &StreamHub{
		StreamID:  streamID,
		Chunks:    make([][]byte, 0, maxChunks),
		MaxCap:    maxChunks,
		Listeners: make(map[*Listener]struct{}),
	}
	h.streams[streamID] = sh
	return sh
}

func (h *Hub) Get(streamID uuid.UUID) *StreamHub {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.streams[streamID]
}

func (h *Hub) Remove(streamID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sh, ok := h.streams[streamID]
	if !ok {
		return
	}

	// Close all listeners
	sh.mu.Lock()
	for l := range sh.Listeners {
		close(l.Ch)
		delete(sh.Listeners, l)
	}
	sh.mu.Unlock()

	delete(h.streams, streamID)
	log.Printf("[hub] removed stream %s", streamID)
}

// --- StreamHub methods ---

func (sh *StreamHub) AddChunk(data []byte) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.Chunks = append(sh.Chunks, data)
	if len(sh.Chunks) > sh.MaxCap {
		sh.Chunks = sh.Chunks[1:]
	}

	// Broadcast to all listeners
	for l := range sh.Listeners {
		select {
		case l.Ch <- data:
		default:
			// Listener too slow, drop
			log.Printf("[hub] dropping chunk for listener %s (slow)", l.ID)
		}
	}
}

func (sh *StreamHub) UpdateState(songID uuid.UUID, revision int64, startedAt int64) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.SongID = songID
	sh.Revision = revision
	sh.StartedAt = startedAt
	// Clear buffer on song change
	sh.Chunks = sh.Chunks[:0]
}

func (sh *StreamHub) Subscribe(l *Listener) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.Listeners[l] = struct{}{}

	// Send latest chunk to new listener
	if len(sh.Chunks) > 0 {
		latest := sh.Chunks[len(sh.Chunks)-1]
		select {
		case l.Ch <- latest:
		default:
		}
	}
}

func (sh *StreamHub) Unsubscribe(l *Listener) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.Listeners, l)
}

// SetCancel stores the cancel function for the current serveStream goroutine.
func (sh *StreamHub) SetCancel(cancel context.CancelFunc) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.cancel = cancel
}

// Cancel stops the current serveStream goroutine.
func (sh *StreamHub) Cancel() {
	sh.mu.RLock()
	cancel := sh.cancel
	sh.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
}
