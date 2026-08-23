package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	PlaylistNameMaxLength = 128
)

type Playlist struct {
	ID        uuid.UUID `json:"id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Name      string    `json:"name"`
	IsPublic  bool      `json:"is_public"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
