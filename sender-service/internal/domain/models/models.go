package models

import (
	"time"

	"github.com/google/uuid"
)

type StreamState struct {
	StreamID       uuid.UUID  `json:"stream_id"`
	CurrentQueueID *uuid.UUID `json:"current_queue_id,omitempty"`
	SongID         *uuid.UUID `json:"song_id,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	IsActive       bool       `json:"is_active"`
	Revision       int64      `json:"revision"`
}

type AudioChunk struct {
	StreamID  uuid.UUID `json:"stream_id"`
	Index     int       `json:"index"`
	Data      []byte    `json:"-"`
	SongID    uuid.UUID `json:"song_id"`
	Revision  int64     `json:"revision"`
}

type StreamEvent struct {
	Type     string    `json:"type"`
	StreamID uuid.UUID `json:"stream_id"`
	Revision int64     `json:"revision"`
}

const (
	EventStreamStarted  = "STREAM_STARTED"
	EventQueueUpdated   = "QUEUE_UPDATED"
	EventSongChanged    = "STREAM_SONG_CHANGED"
	EventStreamEnded    = "STREAM_ENDED"
)
