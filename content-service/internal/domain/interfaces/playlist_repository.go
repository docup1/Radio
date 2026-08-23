package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type PlaylistRepository interface {
	Create(ctx context.Context, p models.Playlist) error
	GetVisible(ctx context.Context, id, viewer uuid.UUID) (*models.Playlist, error)
	ListVisible(ctx context.Context, viewer uuid.UUID, limit, offset int) ([]models.Playlist, error)
	Update(ctx context.Context, id, owner uuid.UUID, patch PlaylistPatch) error
	Delete(ctx context.Context, id, owner uuid.UUID) error
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	OwnerOf(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
}

type PlaylistPatch struct {
	Name     *string
	IsPublic *bool
}
