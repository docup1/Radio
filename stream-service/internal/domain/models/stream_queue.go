package models

import (
	"time"

	"github.com/google/uuid"
)

type StreamQueueItem struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	SongID    uuid.UUID `json:"song_id"`
	Position  int64     `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}
