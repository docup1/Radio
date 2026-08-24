package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type SongRepository interface {
	Create(ctx context.Context, s models.Song) error
	GetVisible(ctx context.Context, id, viewer uuid.UUID) (*models.Song, error)
	ListVisible(ctx context.Context, viewer uuid.UUID, scope models.SongScope, limit, offset int) ([]models.Song, error)
	Update(ctx context.Context, id, owner uuid.UUID, patch SongPatch) error
	Delete(ctx context.Context, id, owner uuid.UUID) error
	SearchVisible(ctx context.Context, q string, viewer uuid.UUID, scope models.SongScope, limit, offset int) ([]models.Song, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	SongByMelody(ctx context.Context, melodyID uuid.UUID) (*models.Song, error)
}

type SongPatch struct {
	Name        *string
	Description *string
	MelodyID    *uuid.UUID
	ImageID     *uuid.NullUUID
	IsPublic    *bool
}
