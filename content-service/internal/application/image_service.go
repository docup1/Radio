package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type ImageService struct {
	Images interfaces.ImageRepository
}

func (s *ImageService) Create(ctx context.Context, in CreateFileInput) (*models.Image, error) {
	if err := validateFileInput(in); err != nil {
		return nil, err
	}
	m := models.Image{
		ID:          uuid.New(),
		Path:        in.Path,
		ContentType: in.ContentType,
		CreatedAt:   time.Now(),
	}
	if err := s.Images.Create(ctx, m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *ImageService) Get(ctx context.Context, id uuid.UUID) (*models.Image, error) {
	return s.Images.Get(ctx, id)
}

func (s *ImageService) List(ctx context.Context, limit, offset int) ([]models.Image, error) {
	return s.Images.List(ctx, limit, offset)
}

func (s *ImageService) Update(ctx context.Context, id uuid.UUID, patch interfaces.ImagePatch) error {
	if patch.Path != nil {
		if len(*patch.Path) == 0 || len(*patch.Path) > models.ImagePathMaxLength {
			return interfaces.ErrInvalid
		}
	}
	if patch.ContentType != nil {
		if len(*patch.ContentType) == 0 || len(*patch.ContentType) > models.ImageContentTypeMaxLength {
			return interfaces.ErrInvalid
		}
	}
	return s.Images.Update(ctx, id, patch)
}

func (s *ImageService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.Images.Delete(ctx, id)
}
