package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type StreamRepository interface {
	Create(ctx context.Context, stream *models.Stream) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Stream, error)
	Update(ctx context.Context, stream *models.Stream) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type StreamStateRepository interface {
	Create(ctx context.Context, state *models.StreamState) error
	GetByStreamID(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error)
	Update(ctx context.Context, state *models.StreamState) error
	Advance(ctx context.Context, streamID uuid.UUID, nextQueueID *uuid.UUID, startedAt time.Time) (*models.StreamState, error)
	SetActive(ctx context.Context, streamID uuid.UUID, active bool) (*models.StreamState, error)
}

type QueueRepository interface {
	Add(ctx context.Context, item *models.StreamQueueItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.StreamQueueItem, error)
	ListByStream(ctx context.Context, streamID uuid.UUID) ([]*models.StreamQueueItem, error)
	GetNextAfterPosition(ctx context.Context, streamID uuid.UUID, position int64) (*models.StreamQueueItem, error)
	GetFirst(ctx context.Context, streamID uuid.UUID) (*models.StreamQueueItem, error)
	GetByStreamAndPosition(ctx context.Context, streamID uuid.UUID, position int64) (*models.StreamQueueItem, error)
	Remove(ctx context.Context, id uuid.UUID) error
	RemoveByStream(ctx context.Context, streamID uuid.UUID) error
	MaxPosition(ctx context.Context, streamID uuid.UUID) (int64, error)
	Count(ctx context.Context, streamID uuid.UUID) (int64, error)
	Move(ctx context.Context, id uuid.UUID, newPosition int64) error
}

type HashtagRepository interface {
	Add(ctx context.Context, hashtag *models.Hashtag) error
	ListByStream(ctx context.Context, streamID uuid.UUID) ([]*models.Hashtag, error)
	Remove(ctx context.Context, id uuid.UUID) error
	RemoveByStream(ctx context.Context, streamID uuid.UUID) error
}

type OutboxRepository interface {
	Insert(ctx context.Context, event *models.OutboxEvent) error
	FetchUnprocessed(ctx context.Context, limit int) ([]*models.OutboxEvent, error)
	MarkProcessed(ctx context.Context, ids []uuid.UUID) error
}
