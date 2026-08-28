package application

import (
	"context"
	"log"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"radio/stream-service/internal/domain/interfaces"
	"radio/stream-service/internal/domain/models"
	svcredis "radio/stream-service/internal/infrastructure/redis"
)

type Repos struct {
	Streams  interfaces.StreamRepository
	Hashtags interfaces.HashtagRepository
}

type Service struct {
	repos   Repos
	q       *svcredis.QueueStore
	pub     *svcredis.Publisher
	rdb     *goredis.Client
	checker interfaces.SongChecker
}

func New(repos Repos, q *svcredis.QueueStore, pub *svcredis.Publisher, checker interfaces.SongChecker, rdb *goredis.Client) *Service {
	return &Service{repos: repos, q: q, pub: pub, checker: checker, rdb: rdb}
}

// --- Stream ---

func (s *Service) GetOrCreateStream(ctx context.Context, userID uuid.UUID) (*models.Stream, error) {
	stream, err := s.repos.Streams.GetByID(ctx, userID)
	if err == nil {
		return stream, nil
	}
	if err != models.ErrNotFound {
		return nil, err
	}

	stream = &models.Stream{
		ID:   userID,
		Name: "My Stream",
	}
	if err := s.repos.Streams.Create(ctx, stream); err != nil {
		return nil, err
	}
	return s.repos.Streams.GetByID(ctx, userID)
}

func (s *Service) GetStream(ctx context.Context, id uuid.UUID) (*models.Stream, error) {
	return s.repos.Streams.GetByID(ctx, id)
}

// ListActiveStreams scans Redis for live streams (feed).
func (s *Service) ListActiveStreams(ctx context.Context) ([]uuid.UUID, error) {
	return svcredis.GetActiveStreamIDs(ctx, s.rdb)
}

func (s *Service) UpdateStream(ctx context.Context, id uuid.UUID, name, description string, loop bool) (*models.Stream, error) {
	stream := &models.Stream{
		ID:          id,
		Name:        name,
		Description: description,
		Loop:        loop,
	}
	if err := s.repos.Streams.Update(ctx, stream); err != nil {
		return nil, err
	}
	// Keep the active-session loop snapshot in sync.
	if err := s.q.SetLoop(ctx, id, loop); err != nil {
		log.Printf("[service] sync loop flag %s: %v", id, err)
	}
	return s.repos.Streams.GetByID(ctx, id)
}

func (s *Service) DeleteStream(ctx context.Context, id uuid.UUID) error {
	if err := s.Stop(ctx, id); err != nil && err != models.ErrConflict {
		return err
	}
	return s.repos.Streams.Delete(ctx, id)
}

// --- Playback control (stream-service owns the state, sender executes) ---

// Start activates a stream: marks it live and tells the sender to serve the first song.
func (s *Service) Start(ctx context.Context, streamID uuid.UUID) error {
	stream, err := s.repos.Streams.GetByID(ctx, streamID)
	if err != nil {
		return err
	}

	active, err := s.q.IsActive(ctx, streamID)
	if err != nil {
		return err
	}
	if active {
		return models.ErrConflict
	}

	items, err := s.q.List(ctx, streamID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return models.ErrInvalid
	}

	first := items[0]
	if err := s.q.SetActive(ctx, streamID); err != nil {
		return err
	}
	if err := s.q.SetCursor(ctx, streamID, first.ID); err != nil {
		return err
	}
	if err := s.q.SetLoop(ctx, streamID, stream.Loop); err != nil {
		return err
	}
	return s.pub.PublishStreamStarted(ctx, streamID, first.ID, first.SongID)
}

// Stop deactivates a stream: wipes all runtime keys and tells the sender to tear down.
func (s *Service) Stop(ctx context.Context, streamID uuid.UUID) error {
	active, err := s.q.IsActive(ctx, streamID)
	if err != nil {
		return err
	}
	if !active {
		return models.ErrConflict
	}

	if err := s.q.Clear(ctx, streamID); err != nil {
		log.Printf("[service] clear %s: %v", streamID, err)
	}
	return s.pub.PublishStreamStopped(ctx, streamID)
}

// --- Status ---

func (s *Service) GetStatus(ctx context.Context, streamID uuid.UUID) (*models.StreamStatus, error) {
	return s.q.Status(ctx, streamID)
}

// --- Queue ---

func (s *Service) AddToQueue(ctx context.Context, streamID, songID uuid.UUID) (*models.QueueItem, error) {
	if _, err := s.repos.Streams.GetByID(ctx, streamID); err != nil {
		return nil, err
	}

	allowed, err := s.checker.Check(ctx, streamID.String(), songID.String())
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, models.ErrForbidden
	}

	item, err := s.q.Add(ctx, streamID, songID)
	if err != nil {
		return nil, err
	}

	if err := s.pub.PublishQueueUpdated(ctx, streamID); err != nil {
		log.Printf("[service] publish queue_updated: %v", err)
	}
	return item, nil
}

func (s *Service) ListQueue(ctx context.Context, streamID uuid.UUID) ([]*models.QueueItem, error) {
	return s.q.List(ctx, streamID)
}

func (s *Service) RemoveFromQueue(ctx context.Context, streamID, itemID uuid.UUID) error {
	if err := s.q.Remove(ctx, streamID, itemID); err != nil {
		return err
	}
	if err := s.pub.PublishQueueUpdated(ctx, streamID); err != nil {
		log.Printf("[service] publish queue_updated (remove): %v", err)
	}
	return nil
}

func (s *Service) MoveQueueItem(ctx context.Context, streamID, itemID uuid.UUID, newPosition int64) error {
	if err := s.q.Move(ctx, streamID, itemID, newPosition); err != nil {
		return err
	}
	if err := s.pub.PublishQueueUpdated(ctx, streamID); err != nil {
		log.Printf("[service] publish queue_updated (move): %v", err)
	}
	return nil
}

// --- Hashtags ---

func (s *Service) AddHashtag(ctx context.Context, streamID uuid.UUID, name string) (*models.Hashtag, error) {
	if len(name) > models.HashtagNameMaxLength {
		return nil, models.ErrInvalid
	}
	h := &models.Hashtag{
		ID:       uuid.New(),
		StreamID: streamID,
		Name:     name,
	}
	if err := s.repos.Hashtags.Add(ctx, h); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *Service) ListHashtags(ctx context.Context, streamID uuid.UUID) ([]*models.Hashtag, error) {
	return s.repos.Hashtags.ListByStream(ctx, streamID)
}

func (s *Service) RemoveHashtag(ctx context.Context, streamID, hashtagID uuid.UUID) error {
	return s.repos.Hashtags.Remove(ctx, hashtagID)
}
