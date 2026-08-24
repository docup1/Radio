package models

import (
	"time"

	"github.com/google/uuid"
)

type StreamState struct {
	StreamID       uuid.UUID  `json:"stream_id"`
	CurrentQueueID *uuid.UUID `json:"current_queue_id,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	IsActive       bool       `json:"is_active"`
	Revision       int64      `json:"revision"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
