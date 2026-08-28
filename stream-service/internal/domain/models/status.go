package models

import "github.com/google/uuid"

// StreamStatus is the runtime status of a stream (Redis-backed).
type StreamStatus struct {
	IsActive       bool       `json:"is_active"`
	CurrentItemID  *uuid.UUID `json:"current_item_id,omitempty"`
	CurrentSongID  *uuid.UUID `json:"current_song_id,omitempty"`
	Position       *int64     `json:"position,omitempty"`
	QueueLength    int64      `json:"queue_length"`
}
