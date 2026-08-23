package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type ImageRepository interface {
	Create(ctx context.Context, m models.Image) error
	Get(ctx context.Context, id uuid.UUID) (*models.Image, error)
	List(ctx context.Context, limit, offset int) ([]models.Image, error)
	Update(ctx context.Context, id uuid.UUID, patch ImagePatch) error
	Delete(ctx context.Context, id uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type ImagePatch struct {
	Path        *string
	ContentType *string
}
