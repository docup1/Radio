package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
	"radio/stream-service/internal/domain/interfaces"
	"radio/stream-service/internal/infrastructure/redis"
)

type Repos struct {
	Streams  interfaces.StreamRepository
	State    interfaces.StreamStateRepository
	Queue    interfaces.QueueRepository
	Hashtags interfaces.HashtagRepository
	Outbox   interfaces.OutboxRepository
}

type Service struct {
	repos Repos
	pub   *redis.Publisher
}

func New(repos Repos, pub *redis.Publisher) *Service {
	return &Service{repos: repos, pub: pub}
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

	// Lazy-create stream with default name
	stream = &models.Stream{
		ID:   userID,
		Name: "My Stream",
	}
	if err := s.repos.Streams.Create(ctx, stream); err != nil {
		return nil, err
	}

	state := &models.StreamState{
		StreamID: userID,
		IsActive: false,
		Revision: 0,
	}
	if err := s.repos.State.Create(ctx, state); err != nil {
		return nil, err
	}
	return s.repos.Streams.GetByID(ctx, userID)
}

func (s *Service) GetStream(ctx context.Context, id uuid.UUID) (*models.Stream, error) {
	return s.repos.Streams.GetByID(ctx, id)
}

func (s *Service) ListActiveStreams(ctx context.Context) ([]*models.StreamState, error) {
	return s.repos.State.ListActive(ctx)
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
	return s.repos.Streams.GetByID(ctx, id)
}

func (s *Service) DeleteStream(ctx context.Context, id uuid.UUID) error {
	return s.repos.Streams.Delete(ctx, id)
}

// --- Queue ---

func (s *Service) AddToQueue(ctx context.Context, streamID, songID uuid.UUID) (*models.StreamQueueItem, error) {
	// Validate stream exists
	if _, err := s.repos.Streams.GetByID(ctx, streamID); err != nil {
		return nil, err
	}

	pos, err := s.repos.Queue.MaxPosition(ctx, streamID)
	if err != nil {
		return nil, err
	}

	item := &models.StreamQueueItem{
		ID:       uuid.New(),
		StreamID: streamID,
		SongID:   songID,
		Position: pos + 1,
	}
	if err := s.repos.Queue.Add(ctx, item); err != nil {
		return nil, err
	}

	// Publish QUEUE_UPDATED via outbox
	if err := s.publishEvent(ctx, "stream", streamID, models.EventQueueUpdated, nil); err != nil {
		log.Printf("[service] outbox QUEUE_UPDATED: %v", err)
	}

	return item, nil
}

func (s *Service) ListQueue(ctx context.Context, streamID uuid.UUID) ([]*models.StreamQueueItem, error) {
	return s.repos.Queue.ListByStream(ctx, streamID)
}

func (s *Service) RemoveFromQueue(ctx context.Context, streamID, itemID uuid.UUID) error {
	item, err := s.repos.Queue.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item.StreamID != streamID {
		return models.ErrNotFound
	}
	return s.repos.Queue.Remove(ctx, itemID)
}

func (s *Service) MoveQueueItem(ctx context.Context, streamID, itemID uuid.UUID, newPosition int64) error {
	item, err := s.repos.Queue.GetByID(ctx, itemID)
	if err != nil {
		return err
	}
	if item.StreamID != streamID {
		return models.ErrNotFound
	}
	return s.repos.Queue.Move(ctx, itemID, newPosition)
}

// --- Stream Control ---

func (s *Service) StartStream(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	_, err := s.repos.Streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	// Get first queue item
	first, err := s.repos.Queue.GetFirst(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if first == nil {
		return nil, models.ErrInvalid
	}

	now := time.Now().UTC()

	// Atomic: set state + outbox
	st, err := s.repos.State.Advance(ctx, streamID, &first.ID, now)
	if err != nil {
		return nil, err
	}
	st.IsActive = true
	if err := s.repos.State.Update(ctx, st); err != nil {
		return nil, err
	}

	// Outbox: STREAM_STARTED
	payload := map[string]interface{}{
		"song_id":    first.SongID.String(),
		"started_at": now.Format(time.RFC3339Nano),
	}
	if err := s.publishEvent(ctx, "stream", streamID, models.EventStreamStarted, payload); err != nil {
		log.Printf("[service] outbox STREAM_STARTED: %v", err)
	}

	return st, nil
}

func (s *Service) StopStream(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	st, err := s.repos.State.SetActive(ctx, streamID, false)
	if err != nil {
		return nil, err
	}
	if err := s.publishEvent(ctx, "stream", streamID, models.EventStreamEnded, nil); err != nil {
		log.Printf("[service] outbox STREAM_ENDED: %v", err)
	}
	return st, nil
}

func (s *Service) GetState(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	return s.repos.State.GetByStreamID(ctx, streamID)
}

func (s *Service) GetQueueItem(ctx context.Context, itemID uuid.UUID) (*models.StreamQueueItem, error) {
	return s.repos.Queue.GetByID(ctx, itemID)
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

// --- Internal: Advance song (called by worker on SONG_COMPLETE or watchdog) ---

func (s *Service) AdvanceSong(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	st, err := s.repos.State.GetByStreamID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if !st.IsActive {
		return st, nil
	}

	// Find next queue item after current
	var next *models.StreamQueueItem
	if st.CurrentQueueID != nil {
		current, err := s.repos.Queue.GetByID(ctx, *st.CurrentQueueID)
		if err != nil {
			return nil, err
		}
		next, err = s.repos.Queue.GetNextAfterPosition(ctx, streamID, current.Position)
		if err != nil {
			return nil, err
		}
	} else {
		next, err = s.repos.Queue.GetFirst(ctx, streamID)
		if err != nil {
			return nil, err
		}
	}

	stream, err := s.repos.Streams.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	if next == nil {
		// No next song
		if stream.Loop {
			// Loop: restart from first
			first, err := s.repos.Queue.GetFirst(ctx, streamID)
			if err != nil {
				return nil, err
			}
			if first == nil {
				// Queue empty → stop
				return s.repos.State.SetActive(ctx, streamID, false)
			}
			st, err = s.repos.State.Advance(ctx, streamID, &first.ID, now)
			if err != nil {
				return nil, err
			}
			payload := map[string]interface{}{
				"song_id":    first.SongID.String(),
				"started_at": now.Format(time.RFC3339Nano),
			}
			if err := s.publishEvent(ctx, "stream", streamID, models.EventSongChanged, payload); err != nil {
				log.Printf("[service] outbox STREAM_SONG_CHANGED: %v", err)
			}
			return st, nil
		}
		// No loop → stop
		st, err = s.repos.State.SetActive(ctx, streamID, false)
		if err != nil {
			return nil, err
		}
		if err := s.publishEvent(ctx, "stream", streamID, models.EventStreamEnded, nil); err != nil {
			log.Printf("[service] outbox STREAM_ENDED: %v", err)
		}
		return st, nil
	}

	// Advance to next song
	st, err = s.repos.State.Advance(ctx, streamID, &next.ID, now)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"song_id":    next.SongID.String(),
		"started_at": now.Format(time.RFC3339Nano),
	}
	if err := s.publishEvent(ctx, "stream", streamID, models.EventSongChanged, payload); err != nil {
		log.Printf("[service] outbox STREAM_SONG_CHANGED: %v", err)
	}
	return st, nil
}

// --- Helpers ---

func (s *Service) publishEvent(ctx context.Context, aggregateType string, aggregateID uuid.UUID, eventType string, payload interface{}) error {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		raw = b
	} else {
		raw = json.RawMessage(`{}`)
	}

	event := &models.OutboxEvent{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       raw,
	}
	return s.repos.Outbox.Insert(ctx, event)
}
