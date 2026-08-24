package application

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"radio/sender-service/internal/domain/models"
)

type SenderConfig struct {
	ContentServiceURL string
	ChunkSize         int64
	BufferSeconds     int
}

type StreamClient interface {
	GetState(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error)
	Advance(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error)
}

type Service struct {
	streamClient StreamClient
	hub          *Hub
	cfg          SenderConfig
	client       *http.Client
	// Per-song ETag cache: songID → etag
	etagCache map[uuid.UUID]string
}

func NewService(streamClient StreamClient, hub *Hub, cfg SenderConfig) *Service {
	return &Service{
		streamClient: streamClient,
		hub:          hub,
		cfg:          cfg,
		client:       &http.Client{Timeout: 30 * time.Second},
		etagCache:    make(map[uuid.UUID]string),
	}
}

// StartStream initializes a hub for the given stream and begins serving audio.
func (s *Service) StartStream(streamID uuid.UUID) {
	log.Printf("[sender] starting stream %s", streamID)
	sh := s.hub.GetOrCreate(streamID, s.cfg.BufferSeconds*10) // ~10 chunks per second

	// Get initial state
	ctx := context.Background()
	st, err := s.streamClient.GetState(ctx, streamID)
	if err != nil {
		log.Printf("[sender] get state %s: %v", streamID, err)
		return
	}

	if st.SongID == nil || st.StartedAt == nil {
		log.Printf("[sender] no active song for stream %s", streamID)
		return
	}

	sh.UpdateState(*st.SongID, st.Revision, st.StartedAt.UnixNano())
	go s.serveStream(streamID, sh)
}

// StopStream removes the hub for the given stream.
func (s *Service) StopStream(streamID uuid.UUID) {
	log.Printf("[sender] stopping stream %s", streamID)
	s.hub.Remove(streamID)
}

// OnSongChanged updates the hub state and restarts chunk serving.
func (s *Service) OnSongChanged(streamID uuid.UUID) {
	sh := s.hub.Get(streamID)
	if sh == nil {
		s.StartStream(streamID)
		return
	}

	ctx := context.Background()
	st, err := s.streamClient.GetState(ctx, streamID)
	if err != nil {
		log.Printf("[sender] get state %s: %v", streamID, err)
		return
	}

	if st.SongID != nil && st.StartedAt != nil {
		// Clear ETag cache for old song
		if sh.SongID != uuid.Nil {
			delete(s.etagCache, sh.SongID)
		}
		sh.UpdateState(*st.SongID, st.Revision, st.StartedAt.UnixNano())
		go s.serveStream(streamID, sh)
	}
}

// serveStream continuously fetches and broadcasts audio chunks.
func (s *Service) serveStream(streamID uuid.UUID, sh *StreamHub) {
	ticker := time.NewTicker(200 * time.Millisecond) // fetch every 200ms
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if sh.SongID == uuid.Nil {
				return
			}

			// Calculate current position
			startedAt := time.Unix(0, sh.StartedAt)
			elapsed := time.Since(startedAt)
			offset := elapsed.Milliseconds() * 16 / 1000 // ~16KB per second for mp3 128kbps

			// Fetch chunk from Content Service
			chunk, err := s.fetchChunk(sh.SongID, offset)
			if err != nil {
				log.Printf("[sender] fetch chunk %s offset=%d: %v", sh.SongID, offset, err)
				continue
			}

			if chunk != nil {
				sh.AddChunk(chunk)
			}
		}
	}
}

// fetchChunk fetches a chunk of audio from Content Service using Range request.
// Supports ETag caching via If-None-Match.
func (s *Service) fetchChunk(songID uuid.UUID, offset int64) ([]byte, error) {
	url := fmt.Sprintf("%s/songs/%s/audio", s.cfg.ContentServiceURL, songID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	end := offset + s.cfg.ChunkSize - 1
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))

	// Add If-None-Match if we have a cached ETag
	if etag, ok := s.etagCache[songID]; ok {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Cache ETag for future requests
	if etag := resp.Header.Get("ETag"); etag != "" {
		s.etagCache[songID] = etag
	}

	// 304 Not Modified — no new data
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
