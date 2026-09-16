package application

import (
	"context"
	"encoding/json"
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
	SongID    uuid.UUID
	StartedAt int64 // unix nanos
	Active    bool  // true while a serveStream goroutine is feeding audio

	// Audio ring buffer (recent chunks for backfill to late joiners)
	Chunks [][]byte
	MaxCap int

	// Playback tracking
	FileSize  int64
	BytesSent int64

	// WebSocket listeners
	Listeners map[*Listener]struct{}
	mu        sync.RWMutex

	cancel context.CancelFunc
}

// OutMsg is a single message on the unified listener channel.
// A message carries either a binary audio chunk (Chunk set) or a text control
// frame (Ctrl set), never both. Using one channel guarantees FIFO ordering
// between control frames and audio chunks, eliminating the boundary race where
// the old two-channel select could write a new song's audio before the "song"
// announce.
type OutMsg struct {
	Chunk []byte // binary audio chunk (mutually exclusive with Ctrl)
	Ctrl  string // text JSON control frame (empty when Chunk is set)
}

type Listener struct {
	ID      string
	Send    chan OutMsg
	OwnerID string // gateway-stamped uid (empty if anonymous)
	Hub     *StreamHub
	Once    bool // sent initial state
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

// Remove tears the entire hub down (used on stream deletion / hard teardown).
func (h *Hub) Remove(streamID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sh, ok := h.streams[streamID]
	if !ok {
		return
	}

	sh.mu.Lock()
	for l := range sh.Listeners {
		close(l.Send)
		delete(sh.Listeners, l)
	}
	sh.mu.Unlock()

	delete(h.streams, streamID)
	log.Printf("[hub] removed stream %s", streamID)
}

// --- StreamHub methods ---

// BeginSong sets up state for a new song and resets the buffer.
func (sh *StreamHub) BeginSong(songID uuid.UUID, startedAt int64) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.SongID = songID
	sh.StartedAt = startedAt
	sh.Active = true
	sh.FileSize = 0
	sh.BytesSent = 0
	sh.Chunks = sh.Chunks[:0]
}

// StopPlayback halts streaming but keeps listeners connected.
func (sh *StreamHub) StopPlayback() {
	sh.mu.Lock()
	sh.Active = false
	sh.Chunks = sh.Chunks[:0]
	sh.FileSize = 0
	sh.BytesSent = 0
	sh.SongID = uuid.Nil
	sh.StartedAt = 0
	cancel := sh.cancel
	sh.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (sh *StreamHub) AddChunk(data []byte) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if !sh.Active {
		return
	}

	sh.Chunks = append(sh.Chunks, data)
	if len(sh.Chunks) > sh.MaxCap {
		sh.Chunks = sh.Chunks[1:]
	}

	for l := range sh.Listeners {
		select {
		case l.Send <- OutMsg{Chunk: data}:
		default:
			log.Printf("[hub] dropping chunk for listener %s (slow)", l.ID)
		}
	}
}

func (sh *StreamHub) Subscribe(l *Listener) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.Listeners[l] = struct{}{}

	// Backfill latest chunk for a new joiner.
	// Both this chunk and the song announce below go through the same ordered
	// channel, so the client always learns about the song before receiving its
	// audio.
	if len(sh.Chunks) > 0 {
		latest := sh.Chunks[len(sh.Chunks)-1]
		select {
		case l.Send <- OutMsg{Chunk: latest}:
		default:
		}
	}

	// Announce current song
	if sh.SongID != uuid.Nil && sh.Active {
		ctrl, _ := json.Marshal(map[string]string{"type": "song", "songId": sh.SongID.String()})
		select {
		case l.Send <- OutMsg{Ctrl: string(ctrl)}:
		default:
		}
	}
}

func (sh *StreamHub) Unsubscribe(l *Listener) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	delete(sh.Listeners, l)
}

func (sh *StreamHub) SetCancel(cancel context.CancelFunc) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.cancel = cancel
}

func (sh *StreamHub) Cancel() {
	sh.mu.RLock()
	cancel := sh.cancel
	sh.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
}

func (sh *StreamHub) RLock() {
	sh.mu.RLock()
}

func (sh *StreamHub) RUnlock() {
	sh.mu.RUnlock()
}

func (sh *StreamHub) Lock() {
	sh.mu.Lock()
}

func (sh *StreamHub) Unlock() {
	sh.mu.Unlock()
}

func (sh *StreamHub) SendControl(data []byte) {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	msg := string(data)
	for l := range sh.Listeners {
		select {
		case l.Send <- OutMsg{Ctrl: msg}:
		default:
		}
	}
}

// SendControlToOne sends a control message to one specific listener.
func (sh *StreamHub) SendControlToOne(l *Listener, data []byte) {
	select {
	case l.Send <- OutMsg{Ctrl: string(data)}:
	default:
	}
}