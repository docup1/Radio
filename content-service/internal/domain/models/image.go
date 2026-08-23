package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	ImagePathMaxLength        = 512
	ImageContentTypeMaxLength = 128
)

type Image struct {
	ID          uuid.UUID `json:"id"`
	Path        string    `json:"path"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
