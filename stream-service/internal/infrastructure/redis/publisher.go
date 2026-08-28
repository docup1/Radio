package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EventMaxLen caps the per-stream events Redis Stream to avoid unbounded growth.
const EventMaxLen = 1000

type Publisher struct {
	rdb *redis.Client
}

func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

// streamEvent is the envelope written to stream:{id}:events.
type streamEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

func (p *Publisher) PublishEvent(ctx context.Context, streamID uuid.UUID, eventType string, payload interface{}) error {
	key := fmt.Sprintf("stream:%s:events", streamID)
	data, err := json.Marshal(streamEvent{Type: eventType, Payload: payload})
	if err != nil {
		return err
	}

	id, err := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: key,
		MaxLen: EventMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd %s: %w", key, err)
	}
	log.Printf("[redis] published %s to %s (id=%s)", eventType, key, id)
	return nil
}

type songPayload struct {
	ItemID string `json:"item_id"`
	SongID string `json:"song_id"`
}

// PublishStreamStarted signals the sender to begin serving the first song.
func (p *Publisher) PublishStreamStarted(ctx context.Context, streamID, itemID, songID uuid.UUID) error {
	return p.PublishEvent(ctx, streamID, "stream_started", songPayload{itemID.String(), songID.String()})
}

// PublishSongChanged signals the sender to switch to another song (e.g. after skip).
func (p *Publisher) PublishSongChanged(ctx context.Context, streamID, itemID, songID uuid.UUID) error {
	return p.PublishEvent(ctx, streamID, "song_changed", songPayload{itemID.String(), songID.String()})
}

// PublishStreamStopped signals the sender to tear the stream down.
func (p *Publisher) PublishStreamStopped(ctx context.Context, streamID uuid.UUID) error {
	return p.PublishEvent(ctx, streamID, "stream_stopped", nil)
}

// PublishQueueUpdated notifies about queue mutations while active.
func (p *Publisher) PublishQueueUpdated(ctx context.Context, streamID uuid.UUID) error {
	return p.PublishEvent(ctx, streamID, "queue_updated", nil)
}
