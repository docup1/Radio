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

	// Audio buffer: last N chunks
	Chunks [][]byte
	MaxCap int

	// Playback tracking
	FileSize  int64
	BytesSent int64

	// Next-song prefetch buffer
	NextSongID   uuid.UUID
	NextChunks   [][]byte
	NextFileSize int64
	prefetchSent bool // true once control msg for next song was sent

	// WebSocket listeners
	Listeners map[*Listener]struct{}
	mu        sync.RWMutex

	cancel context.CancelFunc
}

type Listener struct {
	ID        string
	Ch        chan []byte
	ControlCh chan string
	Hub       *StreamHub
	Once      bool // sent initial state
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

	for l := range sh.Listeners {
		select {
		case l.Ch <- data:
		default:
			log.Printf("[hub] dropping chunk for listener %s (slow)", l.ID)
		}
	}
}

func (sh *StreamHub) UpdateState(songID uuid.UUID, startedAt int64) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.SongID = songID
	sh.StartedAt = startedAt
	sh.FileSize = 0
	sh.BytesSent = 0
	sh.Chunks = sh.Chunks[:0]
}

func (sh *StreamHub) Subscribe(l *Listener) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.Listeners[l] = struct{}{}

	if len(sh.Chunks) > 0 {
		latest := sh.Chunks[len(sh.Chunks)-1]
		select {
		case l.Ch <- latest:
		default:
		}
	}

	// Tell new listeners which song is currently playing
	if sh.SongID != uuid.Nil {
		ctrl, _ := json.Marshal(map[string]string{"type": "song", "songId": sh.SongID.String()})
		select {
		case l.ControlCh <- string(ctrl):
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
		case l.ControlCh <- msg:
		default:
		}
	}
}

// SetNextPrefetch stores pre-fetched chunks for the next song.
func (sh *StreamHub) SetNextPrefetch(songID uuid.UUID, chunks [][]byte, fileSize int64) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.NextSongID = songID
	sh.NextChunks = chunks
	sh.NextFileSize = fileSize
	sh.prefetchSent = false
}

// TakeNextPrefetch returns and clears the pre-fetched next-song data.
// Returns ok=false if nothing was prefetched.
func (sh *StreamHub) TakeNextPrefetch() (uuid.UUID, [][]byte, int64, bool) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.NextSongID == uuid.Nil || len(sh.NextChunks) == 0 {
		return uuid.Nil, nil, 0, false
	}
	songID := sh.NextSongID
	chunks := sh.NextChunks
	fileSize := sh.NextFileSize
	sh.NextSongID = uuid.Nil
	sh.NextChunks = nil
	sh.NextFileSize = 0
	sh.prefetchSent = false
	return songID, chunks, fileSize, true
}

// ClearNextPrefetch discards any prefetched next-song data.
func (sh *StreamHub) ClearNextPrefetch() {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.NextSongID = uuid.Nil
	sh.NextChunks = nil
	sh.NextFileSize = 0
	sh.prefetchSent = false
}
