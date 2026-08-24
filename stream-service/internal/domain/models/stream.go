package models

import (
	"time"

	"github.com/google/uuid"
)

const StreamNameMaxLength = 128

type Stream struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Loop        bool      `json:"loop"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
