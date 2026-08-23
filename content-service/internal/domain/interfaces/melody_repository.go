package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type MelodyRepository interface {
	Create(ctx context.Context, m models.Melody) error
	Get(ctx context.Context, id uuid.UUID) (*models.Melody, error)
	List(ctx context.Context, limit, offset int) ([]models.Melody, error)
	Update(ctx context.Context, id uuid.UUID, patch MelodyPatch) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type MelodyPatch struct {
	Path        *string
	ContentType *string
	Size        *int64
	Hash        *string
}
