package dto

import "github.com/google/uuid"

type CreateStreamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Loop        bool   `json:"loop"`
}

type UpdateStreamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Loop        bool   `json:"loop"`
}

type AddToQueueRequest struct {
	SongID uuid.UUID `json:"song_id"`
}

type MoveQueueRequest struct {
	Position int64 `json:"position"`
}

type AddHashtagRequest struct {
	Name string `json:"name"`
}

type StreamResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Loop        bool      `json:"loop"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type StreamStateResponse struct {
	StreamID       uuid.UUID  `json:"stream_id"`
	CurrentQueueID *uuid.UUID `json:"current_queue_id,omitempty"`
	StartedAt      *string    `json:"started_at,omitempty"`
	IsActive       bool       `json:"is_active"`
	Revision       int64      `json:"revision"`
	UpdatedAt      string     `json:"updated_at"`
}

type QueueItemResponse struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	SongID    uuid.UUID `json:"song_id"`
	Position  int64     `json:"position"`
	CreatedAt string    `json:"created_at"`
}

type HashtagResponse struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	Name      string    `json:"name"`
	CreatedAt string    `json:"created_at"`
}
