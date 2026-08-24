package redis

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const SenderConsumerGroup = "stream-service-sender"

// SenderEvent represents a message from the Sender (e.g., SONG_COMPLETE).
type SenderEvent struct {
	Type     string    `json:"type"`
	StreamID uuid.UUID `json:"stream_id"`
	Revision int64     `json:"revision"`
}

// Subscriber listens to sender:events Redis stream.
type Subscriber struct {
	rdb *redis.Client
}

func NewSubscriber(rdb *redis.Client) *Subscriber {
	// Ensure consumer group exists
	_ = rdb.XGroupCreateMkStream(context.Background(), StreamSenderEvents, SenderConsumerGroup, "0").Err()
	return &Subscriber{rdb: rdb}
}

// ListenBlocks blocks and calls handler for each SONG_COMPLETE event.
// Runs until ctx is cancelled.
func (s *Subscriber) ListenBlocks(ctx context.Context, handler func(SenderEvent)) {
	consumer := "stream-service-worker"

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := s.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    SenderConsumerGroup,
			Consumer: consumer,
			Streams:  []string{StreamSenderEvents, ">"},
			Count:    10,
			Block:    1 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			log.Printf("[redis] readgroup error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				dataStr, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}
				var event SenderEvent
				if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
					log.Printf("[redis] unmarshal sender event: %v", err)
					continue
				}
				handler(event)
			}
		}
	}
}
