package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type UploadSessionRepository interface {
	Create(ctx context.Context, s models.UploadSession) error
	Get(ctx context.Context, id uuid.UUID) (*models.UploadSession, error)
	Update(ctx context.Context, id uuid.UUID, patch UploadSessionPatch) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOwner(ctx context.Context, owner uuid.UUID, limit, offset int) ([]models.UploadSession, error)
}

type UploadSessionPatch struct {
	Status         *string
	ReceivedChunks *int
	FinalPath      *string
	Size           *int64
	Hash           *string
}
