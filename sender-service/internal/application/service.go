package application

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"radio/sender-service/internal/domain/models"
	contenthttp "radio/sender-service/internal/infrastructure/http"
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
	content      *contenthttp.ContentClient
	hub          *Hub
	cfg          SenderConfig
}

func NewService(streamClient StreamClient, content *contenthttp.ContentClient, hub *Hub, cfg SenderConfig) *Service {
	return &Service{
		streamClient: streamClient,
		content:      content,
		hub:          hub,
		cfg:          cfg,
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
	ctx, cancel := context.WithCancel(context.Background())
	sh.SetCancel(cancel)
	go s.serveStream(ctx, streamID, sh)
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
		// Stop previous goroutine
		sh.Cancel()

		// Clear ETag cache for old song
		if sh.SongID != uuid.Nil {
			s.content.ClearETag(sh.SongID.String())
		}
		sh.UpdateState(*st.SongID, st.Revision, st.StartedAt.UnixNano())
		ctx, cancel := context.WithCancel(context.Background())
		sh.SetCancel(cancel)
		go s.serveStream(ctx, streamID, sh)
	}
}

// serveStream continuously fetches and broadcasts audio chunks.
func (s *Service) serveStream(ctx context.Context, streamID uuid.UUID, sh *StreamHub) {
	log.Printf("[sender] serveStream started for stream %s, songID=%s, revision=%d", streamID, sh.SongID, sh.Revision)
	ticker := time.NewTicker(200 * time.Millisecond) // fetch every 200ms
	defer ticker.Stop()
	defer log.Printf("[sender] serveStream stopped for stream %s", streamID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if sh.SongID == uuid.Nil {
				return
			}

			// Calculate current position
			startedAt := time.Unix(0, sh.StartedAt)
			elapsed := time.Since(startedAt)
			offset := elapsed.Milliseconds() * 16 // 128kbps = 16KB/s = 16 bytes/ms

			// Fetch chunk from Content Service (stream ID == owner ID)
			chunk, err := s.content.FetchChunk(sh.SongID.String(), offset, streamID.String())
			if err != nil {
				log.Printf("[sender] fetch chunk %s offset=%d: %v", sh.SongID, offset, err)
				continue
			}

			if chunk != nil {
				sh.AddChunk(chunk)
				log.Printf("[sender] chunk sent stream=%s offset=%d size=%d", streamID, offset, len(chunk))
			}
		}
	}
}
