package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID            uuid.UUID       `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
}

const (
	EventStreamStarted   = "STREAM_STARTED"
	EventQueueUpdated    = "QUEUE_UPDATED"
	EventSongChanged     = "STREAM_SONG_CHANGED"
	EventStreamEnded     = "STREAM_ENDED"
	EventSongComplete    = "SONG_COMPLETE"
)
