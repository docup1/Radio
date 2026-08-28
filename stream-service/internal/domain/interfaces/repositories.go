package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type StreamRepository interface {
	Create(ctx context.Context, stream *models.Stream) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Stream, error)
	Update(ctx context.Context, stream *models.Stream) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type HashtagRepository interface {
	Add(ctx context.Context, hashtag *models.Hashtag) error
	ListByStream(ctx context.Context, streamID uuid.UUID) ([]*models.Hashtag, error)
	Remove(ctx context.Context, id uuid.UUID) error
	RemoveByStream(ctx context.Context, streamID uuid.UUID) error
}

// SongChecker validates that a song exists and belongs to the owner (content-service).
type SongChecker interface {
	Check(ctx context.Context, ownerID, songID string) (bool, error)
}
