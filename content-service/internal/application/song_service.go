package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type SongService struct {
	Songs    interfaces.SongRepository
	Melodies interfaces.MelodyRepository
	Images   interfaces.ImageRepository
}

type CreateSongInput struct {
	Name        string
	Description *string
	MelodyID    uuid.UUID
	ImageID     *uuid.UUID
	IsPublic    bool
}

func (s *SongService) Create(ctx context.Context, owner uuid.UUID, in CreateSongInput) (*models.Song, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > models.SongNameMaxLength {
		return nil, interfaces.ErrInvalid
	}
	if in.Description != nil && len(*in.Description) > models.SongDescriptionMaxLength {
		return nil, interfaces.ErrInvalid
	}
	if exists, err := s.Melodies.Exists(ctx, in.MelodyID); err != nil {
		return nil, err
	} else if !exists {
		return nil, interfaces.ErrInvalid
	}
	var img uuid.NullUUID
	if in.ImageID != nil {
		if exists, err := s.Images.Exists(ctx, *in.ImageID); err != nil {
			return nil, err
		} else if !exists {
			return nil, interfaces.ErrInvalid
		}
		img = uuid.NullUUID{UUID: *in.ImageID, Valid: true}
	}

	song := models.Song{
		ID:          uuid.New(),
		Name:        in.Name,
		Description: deref(in.Description),
		OwnerID:     owner,
		IsPublic:    in.IsPublic,
		MelodyID:    in.MelodyID,
		ImageID:     img,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.Songs.Create(ctx, song); err != nil {
		return nil, err
	}
	return &song, nil
}

func (s *SongService) Get(ctx context.Context, id, viewer uuid.UUID) (*models.Song, error) {
	return s.Songs.GetVisible(ctx, id, viewer)
}

func (s *SongService) List(ctx context.Context, viewer uuid.UUID, limit, offset int) ([]models.Song, error) {
	return s.Songs.ListVisible(ctx, viewer, limit, offset)
}

func (s *SongService) Search(ctx context.Context, q string, viewer uuid.UUID, limit, offset int) ([]models.Song, error) {
	if strings.TrimSpace(q) == "" {
		return nil, interfaces.ErrInvalid
	}
	return s.Songs.SearchVisible(ctx, q, viewer, limit, offset)
}

func (s *SongService) Update(ctx context.Context, id, owner uuid.UUID, patch interfaces.SongPatch) error {
	if patch.Name != nil {
		if strings.TrimSpace(*patch.Name) == "" || len(*patch.Name) > models.SongNameMaxLength {
			return interfaces.ErrInvalid
		}
	}
	if patch.Description != nil && len(*patch.Description) > models.SongDescriptionMaxLength {
		return interfaces.ErrInvalid
	}
	if patch.MelodyID != nil {
		if exists, err := s.Melodies.Exists(ctx, *patch.MelodyID); err != nil {
			return err
		} else if !exists {
			return interfaces.ErrInvalid
		}
	}
	if patch.ImageID != nil && patch.ImageID.Valid {
		if exists, err := s.Images.Exists(ctx, patch.ImageID.UUID); err != nil {
			return err
		} else if !exists {
			return interfaces.ErrInvalid
		}
	}
	return s.Songs.Update(ctx, id, owner, patch)
}

func (s *SongService) Delete(ctx context.Context, id, owner uuid.UUID) error {
	return s.Songs.Delete(ctx, id, owner)
}
