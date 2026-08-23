package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type MelodyService struct {
	Melodies interfaces.MelodyRepository
	Songs    interfaces.SongRepository
}

type CreateFileInput struct {
	Path        string
	ContentType string
}

type AccessResult struct {
	Exists       bool   `json:"exists"`
	HasAccess    bool   `json:"has_access"`
	ContentType  string `json:"content_type"`
}

func validateFileInput(in CreateFileInput) error {
	if strings.TrimSpace(in.Path) == "" || len(in.Path) > models.MelodyPathMaxLength {
		return interfaces.ErrInvalid
	}
	if strings.TrimSpace(in.ContentType) == "" || len(in.ContentType) > models.MelodyContentTypeMaxLength {
		return interfaces.ErrInvalid
	}
	return nil
}

func (s *MelodyService) Create(ctx context.Context, in CreateFileInput) (*models.Melody, error) {
	if err := validateFileInput(in); err != nil {
		return nil, err
	}
	m := models.Melody{
		ID:          uuid.New(),
		Path:        in.Path,
		ContentType: in.ContentType,
		CreatedAt:   time.Now(),
	}
	if err := s.Melodies.Create(ctx, m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *MelodyService) Get(ctx context.Context, id uuid.UUID) (*models.Melody, error) {
	return s.Melodies.Get(ctx, id)
}

func (s *MelodyService) List(ctx context.Context, limit, offset int) ([]models.Melody, error) {
	return s.Melodies.List(ctx, limit, offset)
}

func (s *MelodyService) Update(ctx context.Context, id uuid.UUID, patch interfaces.MelodyPatch) error {
	if patch.Path != nil {
		if strings.TrimSpace(*patch.Path) == "" || len(*patch.Path) > models.MelodyPathMaxLength {
			return interfaces.ErrInvalid
		}
	}
	if patch.ContentType != nil {
		if strings.TrimSpace(*patch.ContentType) == "" || len(*patch.ContentType) > models.MelodyContentTypeMaxLength {
			return interfaces.ErrInvalid
		}
	}
	if patch.Size != nil && *patch.Size < 0 {
		return interfaces.ErrInvalid
	}
	if patch.Hash != nil && len(*patch.Hash) > 64 {
		return interfaces.ErrInvalid
	}
	return s.Melodies.Update(ctx, id, patch)
}

func (s *MelodyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.Melodies.Delete(ctx, id)
}

// Exists reports whether a melody with the given id is stored.
func (s *MelodyService) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.Melodies.Exists(ctx, id)
}

// Access reports whether the given user may use a melody. A melody is
// accessible when it is referenced by a public song or by a song the user owns.
func (s *MelodyService) Access(ctx context.Context, id, user uuid.UUID) (*AccessResult, error) {
	melody, err := s.Melodies.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &AccessResult{Exists: true, ContentType: melody.ContentType}

	song, err := s.Songs.SongByMelody(ctx, id)
	if err != nil {
		if err == interfaces.ErrNotFound {
			res.HasAccess = false
			return res, nil
		}
		return nil, err
	}
	res.HasAccess = song.IsPublic || song.OwnerID == user
	return res, nil
}
