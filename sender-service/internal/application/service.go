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

// OnStreamStarted begins serving a song (called by worker on stream_started/song_changed).
func (s *Service) OnSongChanged(streamID uuid.UUID, songID uuid.UUID) {
	s.hub.GetOrCreate(streamID, s.cfg.BufferSeconds*10)

	sh := s.hub.Get(streamID)
	if sh == nil {
		return
	}

	sh.Cancel()
	sh.ClearNextPrefetch()

	sh.UpdateState(songID, time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	sh.SetCancel(cancel)
	go s.serveStream(ctx, streamID, sh)
}

// StopStream tears the hub for the given stream down.
func (s *Service) StopStream(streamID uuid.UUID) {
	log.Printf("[sender] stopping stream %s", streamID)
	s.StopHeartbeat(streamID)
	s.hub.Remove(streamID)
}

// Skip advances to the next song immediately. Returns the new song ID or nil if
// the stream ended (queue exhausted without loop).
func (s *Service) Skip(streamID uuid.UUID) (*uuid.UUID, error) {
	sh := s.hub.Get(streamID)
	if sh == nil {
		return nil, fmt.Errorf("stream not active")
	}

	ctx := context.Background()
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
		loop, _ := s.rdb.GetLoop(ctx, streamID)
		if loop && len(entries) > 0 {
			next = &entries[0]
		}
	}

	if next == nil {
		log.Printf("[sender] skip auto-stop %s (queue exhausted)", streamID)
		s.StopStream(streamID)
		_ = s.rdb.SetCursor(ctx, streamID, uuid.Nil)
		_ = s.rdb.PublishEvent(ctx, streamID, "stream_ended", nil)
		return nil, nil
	}

	// Cancel the old goroutine BEFORE updating the cursor so that a concurrent
	// handleSongEnd doesn't read the new cursor and race us to auto-stop.
	sh.Cancel()

	if err := s.rdb.SetCursor(ctx, streamID, next.ItemID); err != nil {
		return nil, fmt.Errorf("set cursor: %w", err)
	}
	s.OnSongChanged(streamID, next.SongID)
	songID := next.SongID
	return &songID, nil
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
	log.Printf("[heartbeat] started for stream %s", streamID)
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

// --- Stream Serving ---

// serveStream delivers audio for one song via a producer-consumer pattern:
//   - fetcher goroutine: pulls chunks from content-service into a buffered channel
//   - sender loop: pops from channel and sends to WS clients at real-time pace
//
// The fetcher fills the buffer as fast as possible (no HTTP round-trip gaps).
// The sender pops one chunk per tick (real-time pacing). The fetcher signals
// completion by closing the channel — the sender detects EOF when a tick finds
// the channel closed and empty.
func (s *Service) serveStream(ctx context.Context, streamID uuid.UUID, sh *StreamHub) {
	log.Printf("[sender] serveStream started stream=%s song=%s", streamID, sh.SongID)

	ctrl, _ := json.Marshal(map[string]string{"type": "song", "songId": sh.SongID.String()})
	sh.SendControl(ctrl)

	// Pacing interval: how fast to send chunks to clients
	chunkDuration := float64(s.cfg.ChunkSize) * 8.0 / float64(s.cfg.Bitrate)
	interval := time.Duration(chunkDuration * float64(time.Second))
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	log.Printf("[sender] pacing stream=%s interval=%v bitrate=%d chunkSize=%d",
		streamID, interval, s.cfg.Bitrate, s.cfg.ChunkSize)

	// Buffered channel: fetcher pushes, sender pops on tick
	chunkCh := make(chan []byte, s.cfg.PrefetchCount)
	sizeCh := make(chan int64, 1)

	// Start fetcher — closes chunkCh when done
	go s.fetchChunks(ctx, sh.SongID.String(), streamID.String(), chunkCh, sizeCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer log.Printf("[sender] serveStream stopped stream=%s", streamID)

	// Wait for file size from first fetch before starting paced delivery
	fileSize := int64(0)
	select {
	case <-ctx.Done():
		return
	case fs := <-sizeCh:
		fileSize = fs
		sh.mu.Lock()
		sh.FileSize = fs
		sh.mu.Unlock()
	}

	bytesSent := int64(0)
	prefetchTriggered := false

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// Pop one chunk per tick — this IS the pacing
			chunk, ok := <-chunkCh
			if !ok {
				log.Printf("[sender] EOF song %s stream %s (%d bytes)", sh.SongID, streamID, bytesSent)
				s.handleSongTransition(ctx, streamID, sh)
				return
			}

			sh.AddChunk(chunk)
			bytesSent += int64(len(chunk))
			sh.mu.Lock()
			sh.BytesSent = bytesSent
			sh.mu.Unlock()

			log.Printf("[sender] chunk stream=%s song=%s chunk=%d/%d total=%d/%d",
				streamID, sh.SongID, len(chunk), s.cfg.ChunkSize, bytesSent, fileSize)

			// At 80% → prefetch next song
			if !prefetchTriggered && fileSize > 0 && bytesSent >= fileSize*80/100 {
				prefetchTriggered = true
				go s.prefetchNextSong(streamID, sh)
			}
		}
	}
}

// fetchChunks pulls chunks from content-service as fast as possible and pushes
// them into chunkCh. Sends file size on sizeCh on first response. Closes chunkCh
// when done (EOF or fatal error) — the sender detects EOF through channel closure.
func (s *Service) fetchChunks(ctx context.Context, songID, ownerID string, chunkCh chan<- []byte, sizeCh chan<- int64) {
	defer close(chunkCh)

	offset := int64(0)
	first := true
	consecutiveErrors := 0

	for {
		if ctx.Err() != nil {
			return
		}

		result, err := s.content.FetchChunk(songID, offset, ownerID)
		if err != nil {
			consecutiveErrors++
			log.Printf("[sender] fetch chunk %s offset=%d: %v (errors=%d)", songID, offset, err, consecutiveErrors)
			if consecutiveErrors >= 5 {
				log.Printf("[sender] too many fetch errors, closing stream")
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		consecutiveErrors = 0

		// Empty response = EOF
		if result == nil || len(result.Data) == 0 {
			if result != nil && result.FileSize > 0 && first {
				select {
				case sizeCh <- result.FileSize:
				default:
				}
			}
			return
		}

		// Signal file size on first response
		if first && result.FileSize > 0 {
			select {
			case sizeCh <- result.FileSize:
			default:
			}
			first = false
		}

		// Push chunk to buffer (blocks if buffer is full — natural backpressure)
		select {
		case chunkCh <- result.Data:
		case <-ctx.Done():
			return
		}

		offset += int64(len(result.Data))

		// Short read = EOF
		if int64(len(result.Data)) < s.cfg.ChunkSize {
			return
		}
	}
}

// fetchChunksFrom is like fetchChunks but starts from a given offset and
// does not signal sizeCh (file size is already known).
func (s *Service) fetchChunksFrom(ctx context.Context, songID, ownerID string, chunkCh chan<- []byte, startOffset int64) {
	offset := startOffset
	consecutiveErrors := 0

	for {
		if ctx.Err() != nil {
			return
		}

		result, err := s.content.FetchChunk(songID, offset, ownerID)
		if err != nil {
			consecutiveErrors++
			log.Printf("[sender] fetch chunk %s offset=%d: %v (errors=%d)", songID, offset, err, consecutiveErrors)
			if consecutiveErrors >= 5 {
				log.Printf("[sender] too many fetch errors, closing stream")
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		consecutiveErrors = 0

		if result == nil || len(result.Data) == 0 {
			return
		}

		select {
		case chunkCh <- result.Data:
		case <-ctx.Done():
			return
		}

		offset += int64(len(result.Data))

		if int64(len(result.Data)) < s.cfg.ChunkSize {
			return
		}
	}
}

// handleSongTransition checks for a prefetched next song and starts it
// immediately, or falls back to handleSongEnd.
func (s *Service) handleSongTransition(ctx context.Context, streamID uuid.UUID, sh *StreamHub) {
	if ctx.Err() != nil {
		return
	}

	// Fast path: next song was prefetched
	if songID, chunks, fileSize, ok := sh.TakeNextPrefetch(); ok {
		log.Printf("[sender] transition (prefetch hit) stream=%s next=%s chunks=%d", streamID, songID, len(chunks))

		// Set cursor for next song
		entries, err := s.rdb.SnapshotQueue(ctx, streamID)
		if err != nil {
			log.Printf("[sender] snapshot queue %s: %v", streamID, err)
		}
		cursor, err := s.rdb.GetCursor(ctx, streamID)
		if err != nil {
			log.Printf("[sender] get cursor %s: %v", streamID, err)
		}
		next := nextEntry(entries, cursor)
		if next != nil {
			_ = s.rdb.SetCursor(ctx, streamID, next.ItemID)
		}

		sh.Cancel()
		sh.UpdateState(songID, time.Now().UnixNano())

		// Send song control
		ctrl, _ := json.Marshal(map[string]string{"type": "song", "songId": songID.String()})
		sh.SendControl(ctrl)

		// Feed prefetched chunks into a new serveStream via a preloaded channel
		go s.serveStreamPreloaded(ctx, streamID, sh, songID, fileSize, chunks)
		return
	}

	// Slow path: no prefetch, use existing handleSongEnd
	s.handleSongEnd(ctx, streamID)
}

// serveStreamPreloaded starts a song with prefetched chunks already in memory.
// It feeds them into a channel, then starts fetchChunks for remaining chunks.
func (s *Service) serveStreamPreloaded(ctx context.Context, streamID uuid.UUID, sh *StreamHub, songID uuid.UUID, fileSize int64, preloaded [][]byte) {
	log.Printf("[sender] serveStreamPreloaded stream=%s song=%s preloaded=%d", streamID, songID, len(preloaded))

	ctrl, _ := json.Marshal(map[string]string{"type": "song", "songId": songID.String()})
	sh.SendControl(ctrl)

	// Pacing interval
	chunkDuration := float64(s.cfg.ChunkSize) * 8.0 / float64(s.cfg.Bitrate)
	interval := time.Duration(chunkDuration * float64(time.Second))
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	chunkCh := make(chan []byte, s.cfg.PrefetchCount)

	// Single producer goroutine: push preloaded chunks, then fetch remaining
	go func() {
		defer close(chunkCh)
		for _, c := range preloaded {
			select {
			case chunkCh <- c:
			case <-ctx.Done():
				return
			}
		}
		// Now fetch remaining chunks starting after preloaded offset
		startOffset := int64(len(preloaded)) * s.cfg.ChunkSize
		s.fetchChunksFrom(ctx, songID.String(), streamID.String(), chunkCh, startOffset)
	}()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Set file size
	sh.mu.Lock()
	sh.FileSize = fileSize
	sh.mu.Unlock()

	bytesSent := int64(len(preloaded)) * s.cfg.ChunkSize
	prefetchTriggered := len(preloaded) > 0

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			chunk, ok := <-chunkCh
			if !ok {
				log.Printf("[sender] EOF song %s stream %s (preloaded, %d bytes)", songID, streamID, bytesSent)
				s.handleSongTransition(ctx, streamID, sh)
				return
			}
			sh.AddChunk(chunk)
			bytesSent += int64(len(chunk))
			sh.mu.Lock()
			sh.BytesSent = bytesSent
			sh.mu.Unlock()

			log.Printf("[sender] chunk stream=%s song=%s chunk=%d/%d total=%d/%d",
				streamID, songID, len(chunk), s.cfg.ChunkSize, bytesSent, fileSize)

			if !prefetchTriggered && fileSize > 0 && bytesSent >= fileSize*80/100 {
				prefetchTriggered = true
				go s.prefetchNextSong(streamID, sh)
			}
		}
	}
}

// prefetchNextSong loads the first chunk(s) of the next song in the queue
// so that the transition is seamless.
func (s *Service) prefetchNextSong(streamID uuid.UUID, sh *StreamHub) {
	ctx := context.Background()

	entries, err := s.rdb.SnapshotQueue(ctx, streamID)
	if err != nil {
		log.Printf("[sender] prefetch snapshot %s: %v", streamID, err)
		return
	}
	cursor, err := s.rdb.GetCursor(ctx, streamID)
	if err != nil {
		log.Printf("[sender] prefetch cursor %s: %v", streamID, err)
		return
	}

	next := nextEntry(entries, cursor)
	if next == nil {
		// Try loop
		loop, _ := s.rdb.GetLoop(ctx, streamID)
		if loop && len(entries) > 0 {
			next = &entries[0]
		}
	}
	if next == nil {
		return
	}

	// Fetch first N chunks of next song
	n := s.cfg.NextSongPrefetch
	if n <= 0 {
		n = 1
	}
	var chunks [][]byte
	var fileSize int64
	offset := int64(0)

	for i := 0; i < n; i++ {
		result, err := s.content.FetchChunk(next.SongID.String(), offset, streamID.String())
		if err != nil {
			log.Printf("[sender] prefetch chunk %s offset=%d: %v", next.SongID, offset, err)
			break
		}
		if result == nil || len(result.Data) == 0 {
			break
		}
		if result.FileSize > 0 {
			fileSize = result.FileSize
		}
		chunks = append(chunks, result.Data)
		offset += int64(len(result.Data))

		if int64(len(result.Data)) < s.cfg.ChunkSize {
			break // short read = EOF
		}
	}

	if len(chunks) > 0 {
		sh.SetNextPrefetch(next.SongID, chunks, fileSize)
		log.Printf("[sender] prefetch ready stream=%s next=%s chunks=%d", streamID, next.SongID, len(chunks))
	}
}

// handleSongEnd advances playback after EOF. The sender owns the transition:
// it reads queue + cursor + loop directly from Redis and either starts the next
// song or auto-stops when the queue is exhausted.
func (s *Service) handleSongEnd(ctx context.Context, streamID uuid.UUID) {
	// If the context was cancelled (e.g. by a concurrent Skip), bail out —
	// another goroutine is taking over.
	if ctx.Err() != nil {
		return
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
		loop, err := s.rdb.GetLoop(ctx, streamID)
		if err != nil {
			log.Printf("[sender] get loop %s: %v", streamID, err)
		}
		if loop && len(entries) > 0 {
			next = &entries[0]
		}
	}

	if next == nil {
		// Queue exhausted: stop playback but keep queue so user can restart.
		log.Printf("[sender] auto-stop %s (queue exhausted)", streamID)
		s.StopStream(streamID)
		_ = s.rdb.SetCursor(ctx, streamID, uuid.Nil)
		_ = s.rdb.PublishEvent(ctx, streamID, "stream_ended", nil)
		return
	}

	if err := s.rdb.SetCursor(ctx, streamID, next.ItemID); err != nil {
		log.Printf("[sender] set cursor %s: %v", streamID, err)
		return
	}
	s.OnSongChanged(streamID, next.SongID)
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
