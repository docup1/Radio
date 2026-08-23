package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	SongNameMaxLength        = 128
	SongDescriptionMaxLength = 512
)

type Song struct {
	ID          uuid.UUID      `json:"id"`
	OwnerID     uuid.UUID      `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	MelodyID    uuid.UUID      `json:"melody_id"`
	ImageID     uuid.NullUUID  `json:"image_id"`
	IsPublic    bool           `json:"is_public"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
