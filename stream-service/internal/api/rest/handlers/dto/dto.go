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

type StreamStatusResponse struct {
	IsActive      bool       `json:"is_active"`
	CurrentItemID *uuid.UUID `json:"current_item_id,omitempty"`
	CurrentSongID *uuid.UUID `json:"current_song_id,omitempty"`
	Position      *int64     `json:"position,omitempty"`
	QueueLength   int64      `json:"queue_length"`
}

type QueueItemResponse struct {
	ID       uuid.UUID `json:"id"`
	SongID   uuid.UUID `json:"song_id"`
	Position int64     `json:"position"`
}

type HashtagResponse struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	Name      string    `json:"name"`
	CreatedAt string    `json:"created_at"`
}
