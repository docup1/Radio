package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	MelodyPathMaxLength        = 512
	MelodyContentTypeMaxLength = 128
)

type Melody struct {
	ID          uuid.UUID `json:"id"`
	Path        string    `json:"path"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
