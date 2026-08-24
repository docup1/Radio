package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type PlaylistService struct {
	Playlists interfaces.PlaylistRepository
}

type CreatePlaylistInput struct {
	Name string
}

func (s *PlaylistService) Create(ctx context.Context, owner uuid.UUID, in CreatePlaylistInput) (*models.Playlist, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > models.PlaylistNameMaxLength {
		return nil, interfaces.ErrInvalid
	}
	p := models.Playlist{
		ID:        uuid.New(),
		Name:      in.Name,
		OwnerID:   owner,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.Playlists.Create(ctx, p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *PlaylistService) Get(ctx context.Context, id, viewer uuid.UUID) (*models.Playlist, error) {
	return s.Playlists.GetVisible(ctx, id, viewer)
}

func (s *PlaylistService) List(ctx context.Context, viewer uuid.UUID, limit, offset int) ([]models.Playlist, error) {
	return s.Playlists.ListVisible(ctx, viewer, limit, offset)
}

func (s *PlaylistService) Update(ctx context.Context, id, owner uuid.UUID, patch interfaces.PlaylistPatch) error {
	if patch.Name != nil {
		if strings.TrimSpace(*patch.Name) == "" || len(*patch.Name) > models.PlaylistNameMaxLength {
			return interfaces.ErrInvalid
		}
	}
	return s.Playlists.Update(ctx, id, owner, patch)
}

func (s *PlaylistService) Delete(ctx context.Context, id, owner uuid.UUID) error {
	return s.Playlists.Delete(ctx, id, owner)
}
