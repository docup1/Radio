package models

import "github.com/google/uuid"

// QueueItem is a single entry in the runtime queue (Redis-backed, ephemeral).
type QueueItem struct {
	ID       uuid.UUID `json:"id"`
	SongID   uuid.UUID `json:"song_id"`
	Position int64     `json:"position"`
}
