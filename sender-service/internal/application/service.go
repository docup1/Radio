package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	contenthttp "radio/sender-service/internal/infrastructure/http"
	"radio/sender-service/internal/infrastructure/redis"
	"radio/sender-service/internal/media"
)

const heartbeatInterval = 10 * time.Second

type SenderConfig struct {
	ContentServiceURL string
	ChunkSize         int64
	Bitrate           int64
	BufferSeconds     int
	PrefetchCount     int
	NextSongPrefetch  int
}

type Service struct {
	content *contenthttp.ContentClient
	rdb     *redis.Client
	hub     *Hub
	cfg     SenderConfig

	opMu        sync.Mutex // serializes control ops (start/stop/skip) and auto-end
	heartbeatMu sync.Mutex
	heartbeats  map[uuid.UUID]context.CancelFunc
}

func NewService(content *contenthttp.ContentClient, rdb *redis.Client, hub *Hub, cfg SenderConfig) *Service {
	return &Service{
		content:    content,
		rdb:        rdb,
		hub:        hub,
		cfg:        cfg,
		heartbeats: make(map[uuid.UUID]context.CancelFunc),
	}
}

// --- WebSocket control (called from the WS handler, owner only) ---

// Start begins playback from the head of the queue.
func (s *Service) Start(streamID uuid.UUID) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	ctx := context.Background()
	active, err := s.rdb.IsActive(ctx, streamID)
	if err != nil {
		return fmt.Errorf("is_active: %w", err)
	}
	if active {
		return fmt.Errorf("stream already active")
	}

	entries, err := s.rdb.SnapshotQueue(ctx, streamID)
	if err != nil {
		return fmt.Errorf("snapshot queue: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("queue is empty")
	}

	first := entries[0]
	if err := s.rdb.SetActive(ctx, streamID); err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	if err := s.rdb.SetCursor(ctx, streamID, first.ItemID); err != nil {
		return fmt.Errorf("set cursor: %w", err)
	}

	log.Printf("[sender] start stream=%s first_song=%s", streamID, first.SongID)
	s.StartHeartbeat(streamID)
	s.OnSongChanged(streamID, first.SongID)
	s.BroadcastState(streamID)
	return nil
}

// Stop halts playback but keeps the queue and listeners' sockets open.
func (s *Service) Stop(streamID uuid.UUID) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	ctx := context.Background()
	active, err := s.rdb.IsActive(ctx, streamID)
	if err != nil {
		return fmt.Errorf("is_active: %w", err)
	}
	if !active {
		return fmt.Errorf("stream is not active")
	}

	s.StopHeartbeat(streamID)
	if err := s.rdb.Deactivate(ctx, streamID); err != nil {
		log.Printf("[sender] deactivate %s: %v", streamID, err)
	}
	s.stopPlayback(streamID, true)
	s.BroadcastState(streamID)
	return nil
}

// Skip advances to the next song immediately. Returns the new song ID or nil
// if the queue is exhausted.
func (s *Service) Skip(streamID uuid.UUID) (*uuid.UUID, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	ctx := context.Background()
	active, err := s.rdb.IsActive(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("is_active: %w", err)
	}
	if !active {
		return nil, fmt.Errorf("stream is not active")
	}

	sh := s.hub.Get(streamID)
	if sh == nil {
		return nil, fmt.Errorf("stream not ready")
	}

	sh.RLock()
	prev := sh.SongID
	sh.RUnlock()

	entries, err := s.rdb.SnapshotQueue(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("snapshot queue: %w", err)
	}
	cursor, err := s.rdb.GetCursor(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("get cursor: %w", err)
	}

	next := nextEntry(entries, cursor)

	if next == nil {
		log.Printf("[sender] skip auto-stop %s (queue exhausted)", streamID)
		ctrl := []byte(`{"type":"stream_ended","message":"Очередь закончилась"}`)
		sh.SendControl(ctrl)
		s.finishStop(streamID)
		return nil, nil
	}

	// Cancel the old goroutine BEFORE updating the cursor so a concurrent
	// handleSongEnd cannot race us to auto-stop.
	sh.Cancel()

	if prev != uuid.Nil {
		ctrl, _ := json.Marshal(songEvent("song_ended", prev))
		sh.SendControl(ctrl)
	}

	if err := s.rdb.SetCursor(ctx, streamID, next.ItemID); err != nil {
		return nil, fmt.Errorf("set cursor: %w", err)
	}
	s.OnSongChanged(streamID, next.SongID)
	s.BroadcastState(streamID)
	return &next.SongID, nil
}

// HandleStreamStopped runs when the worker observes a stream_stopped event
// (published by stream-service REST stop). Runtime keys are already cleared
// there; here we only stop playback and notify listeners.
func (s *Service) HandleStreamStopped(streamID uuid.UUID) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.StopHeartbeat(streamID)
	s.stopPlayback(streamID, true)
	s.BroadcastState(streamID)
}

// --- Playback primitives ---

// OnSongChanged announces a (new) song and starts serving its audio.
func (s *Service) OnSongChanged(streamID uuid.UUID, songID uuid.UUID) {
	log.Printf("[sender] OnSongChanged stream=%s song=%s", streamID, songID)
	sh := s.hub.GetOrCreate(streamID, s.maxChunks())

	sh.Cancel()
	sh.BeginSong(songID, time.Now().UnixNano())

	ctrl, _ := json.Marshal(songEvent("song", songID))
	sh.SendControl(ctrl)

	ctx, cancel := context.WithCancel(context.Background())
	sh.SetCancel(cancel)
	go s.serveStream(ctx, streamID, sh)
}

// stopPlayback halts streaming for the stream but keeps the hub (and its
// listeners) alive. The caller must hold opMu.
func (s *Service) stopPlayback(streamID uuid.UUID, emitStopped bool) {
	sh := s.hub.Get(streamID)
	if sh == nil {
		sh = s.hub.GetOrCreate(streamID, s.maxChunks())
	}
	if emitStopped {
		sh.SendControl([]byte(`{"type":"stream_stopped"}`))
	}
	sh.StopPlayback()
}

// finishStop deactivates redis state and stops playback after a stream ended.
// The caller must hold opMu.
func (s *Service) finishStop(streamID uuid.UUID) {
	s.StopHeartbeat(streamID)
	if err := s.rdb.Deactivate(context.Background(), streamID); err != nil {
		log.Printf("[sender] deactivate %s: %v", streamID, err)
	}
	s.stopPlayback(streamID, false)
	s.BroadcastState(streamID)
}

// serveStream continuously delivers the current song as MPEG-frame-aligned
// buffers. Delivery is paced to the song's real playback timeline (each chunk
// is sent slightly before it needs to play), so the browser receives audio in
// real time while keeping a small lead buffer against network jitter.
func (s *Service) serveStream(ctx context.Context, streamID uuid.UUID, sh *StreamHub) {
	songID := sh.SongID
	log.Printf("[sender] serveStream started stream=%s song=%s", streamID, songID)

	parser := media.NewMP3Parser()
	var pending []byte
	songStart := time.Now()

	// pacingLead is how far ahead of its playback position a chunk is
	// delivered. Kept small so song changes and stream stop stay responsive.
	const pacingLead = 800 * time.Millisecond

	// emittedDur is the playback position where the next chunk's audio begins
	// (== the total duration of all audio emitted so far). A chunk covering
	// audio at position P is sent at wall-clock songStart+P-lead, so the client
	// keeps the whole chunk as lead after the first one.
	emittedDur := time.Duration(0)
	// pendingDur is the duration of the frames currently buffered in pending.
	pendingDur := time.Duration(0)

	emit := func(force bool) {
		if len(pending) == 0 || (!force && int64(len(pending)) < s.cfg.ChunkSize) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if !force {
			if d := time.Until(songStart.Add(emittedDur - pacingLead)); d > 0 {
				log.Printf("[sender] pacing stream=%s song=%s at=%s sleep=%s", streamID, songID, emittedDur.Round(time.Millisecond), d.Round(time.Millisecond))
				select {
				case <-ctx.Done():
					return
				case <-time.After(d):
				}
			}
		}
		sh.AddChunk(pending)
		sh.mu.Lock()
		sh.BytesSent += int64(len(pending))
		sh.mu.Unlock()
		emittedDur += pendingDur
		pendingDur = 0
		pending = nil
	}

	offset := int64(0)
	consecutiveErrors := 0

fetch:
	for {
		if ctx.Err() != nil {
			log.Printf("[sender] serveStream canceled stream=%s song=%s", streamID, songID)
			return
		}

		result, err := s.content.FetchChunk(songID.String(), offset, streamID.String())
		if err != nil {
			consecutiveErrors++
			log.Printf("[sender] fetch chunk %s offset=%d: %v (errors=%d)", songID, offset, err, consecutiveErrors)
			if consecutiveErrors >= 5 {
				log.Printf("[sender] too many fetch errors, ending song %s", songID)
				break
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		consecutiveErrors = 0

		if result.FileSize > 0 {
			sh.mu.Lock()
			sh.FileSize = result.FileSize
			sh.mu.Unlock()
		}

		// Empty response = EOF
		if result == nil || len(result.Data) == 0 {
			break fetch
		}

		frames := parser.Feed(result.Data)
		for _, f := range frames {
			pending = append(pending, f...)
			pendingDur += media.FrameDuration(f)
			emit(false)
		}

		offset += int64(len(result.Data))

		// Short read = EOF
		if int64(len(result.Data)) < s.cfg.ChunkSize {
			break fetch
		}
	}

	if ctx.Err() != nil {
		log.Printf("[sender] serveStream canceled stream=%s song=%s", streamID, songID)
		return
	}

	emit(true)

	// Final partial frame (stream may not end on a boundary)
	if tail := parser.Flush(); len(tail) > 0 {
		pending = append(pending, tail...)
		emit(true)
	}

	log.Printf("[sender] EOF song %s stream %s (%d bytes)", songID, streamID, sh.BytesSent)
	s.handleSongEnd(ctx, streamID, sh)
}

// handleSongEnd emits song_ended and then advances to the next song or
// auto-stops when the queue is exhausted.
func (s *Service) handleSongEnd(ctx context.Context, streamID uuid.UUID, sh *StreamHub) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	if ctx.Err() != nil {
		return
	}

	sh.RLock()
	ended := sh.SongID
	sh.RUnlock()
	if ended != uuid.Nil {
		ctrl, _ := json.Marshal(songEvent("song_ended", ended))
		sh.SendControl(ctrl)
	}

	entries, err := s.rdb.SnapshotQueue(ctx, streamID)
	if err != nil {
		log.Printf("[sender] snapshot queue %s: %v", streamID, err)
		return
	}
	cursor, err := s.rdb.GetCursor(ctx, streamID)
	if err != nil {
		log.Printf("[sender] get cursor %s: %v", streamID, err)
		return
	}

	next := nextEntry(entries, cursor)
	if next == nil {
		log.Printf("[sender] auto-stop %s (queue exhausted)", streamID)
		sh.SendControl([]byte(`{"type":"stream_ended","message":"Очередь закончилась"}`))
		s.finishStop(streamID)
		return
	}

	if err := s.rdb.SetCursor(ctx, streamID, next.ItemID); err != nil {
		log.Printf("[sender] set cursor %s: %v", streamID, err)
		return
	}
	s.OnSongChanged(streamID, next.SongID)
	s.BroadcastState(streamID)
}

// --- State broadcast ---

type streamStatus struct {
	IsActive      bool    `json:"is_active"`
	CurrentSongID *string `json:"current_song_id,omitempty"`
	CurrentItemID *string `json:"current_item_id,omitempty"`
	Position      *int64  `json:"position,omitempty"`
	QueueLength   int64   `json:"queue_length"`
}

// Status builds the runtime snapshot for the stream from redis + hub.
func (s *Service) Status(ctx context.Context, streamID uuid.UUID) (*streamStatus, error) {
	st := &streamStatus{}

	entries, err := s.rdb.SnapshotQueue(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("snapshot queue: %w", err)
	}
	st.QueueLength = int64(len(entries))

	active, err := s.rdb.IsActive(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("is_active: %w", err)
	}
	st.IsActive = active
	if !active {
		return st, nil
	}

	cursor, err := s.rdb.GetCursor(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("get cursor: %w", err)
	}
	for i, e := range entries {
		if e.ItemID == cursor {
			cid, sid, pos := e.ItemID.String(), e.SongID.String(), int64(i)
			st.CurrentItemID = &cid
			st.CurrentSongID = &sid
			st.Position = &pos
			break
		}
	}
	return st, nil
}

// BroadcastState pushes a state message to every listener of the stream.
func (s *Service) BroadcastState(streamID uuid.UUID) {
	st, err := s.Status(context.Background(), streamID)
	if err != nil {
		log.Printf("[sender] status %s: %v", streamID, err)
		return
	}
	data, _ := json.Marshal(map[string]interface{}{"type": "state", "data": st})
	if sh := s.hub.Get(streamID); sh != nil {
		sh.SendControl(data)
	}
}

// SendStateTo sends a state snapshot to a single listener (on connect).
func (s *Service) SendStateTo(streamID uuid.UUID, l *Listener) {
	st, err := s.Status(context.Background(), streamID)
	if err != nil {
		log.Printf("[sender] status %s: %v", streamID, err)
		return
	}
	data, _ := json.Marshal(map[string]interface{}{"type": "state", "data": st})
	if sh := s.hub.Get(streamID); sh != nil {
		sh.SendControlToOne(l, data)
	}
}

// songEvent builds a {"type":...,"songId":...} control message.
func songEvent(t string, songID uuid.UUID) map[string]string {
	return map[string]string{"type": t, "songId": songID.String()}
}

func (s *Service) maxChunks() int {
	n := s.cfg.BufferSeconds * 10
	if n < 16 {
		n = 16
	}
	return n
}

// --- Heartbeat ---

func (s *Service) StartHeartbeat(streamID uuid.UUID) {
	s.heartbeatMu.Lock()
	defer s.heartbeatMu.Unlock()

	if cancel, ok := s.heartbeats[streamID]; ok {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.heartbeats[streamID] = cancel

	go s.heartbeatLoop(ctx, streamID)
}

func (s *Service) StopHeartbeat(streamID uuid.UUID) {
	s.heartbeatMu.Lock()
	defer s.heartbeatMu.Unlock()

	if cancel, ok := s.heartbeats[streamID]; ok {
		cancel()
		delete(s.heartbeats, streamID)
	}
}

func (s *Service) heartbeatLoop(ctx context.Context, streamID uuid.UUID) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	defer log.Printf("[heartbeat] stopped for stream %s", streamID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.rdb.RefreshActive(ctx, streamID); err != nil {
				log.Printf("[heartbeat] refresh %s: %v", streamID, err)
			}
		}
	}
}

// nextEntry finds the entry following cursor. Falls back to the head when the
// cursor entry no longer exists; returns nil at the end of the queue.
func nextEntry(entries []redis.QueueEntry, cursor uuid.UUID) *redis.QueueEntry {
	idx := -1
	for i, e := range entries {
		if e.ItemID == cursor {
			idx = i
			break
		}
	}
	if idx < 0 {
		if len(entries) > 0 {
			return &entries[0]
		}
		return nil
	}
	if idx+1 < len(entries) {
		return &entries[idx+1]
	}
	return nil
}
