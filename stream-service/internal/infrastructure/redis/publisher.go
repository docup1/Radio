package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"radio/stream-service/internal/infrastructure"
)

const (
	StreamEvents       = "stream:events"
	StreamSenderEvents = "stream:sender:events"
	ConsumerGroup      = "stream-service"
)

func NewClient(cfg infrastructure.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// Ensure consumer group for stream:events
	err := rdb.XGroupCreateMkStream(ctx, StreamEvents, ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("xgroup create %s: %w", StreamEvents, err)
	}

	return rdb, nil
}

type Publisher struct {
	rdb *redis.Client
}

func NewPublisher(rdb *redis.Client) *Publisher {
	return &Publisher{rdb: rdb}
}

type StreamEvent struct {
	Type     string      `json:"type"`
	StreamID uuid.UUID   `json:"stream_id"`
	Revision int64       `json:"revision"`
	Payload  interface{} `json:"payload,omitempty"`
}

func (p *Publisher) Publish(ctx context.Context, event StreamEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	id, err := p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamEvents,
		Values: map[string]interface{}{
			"type":   event.Type,
			"stream": event.StreamID.String(),
			"data":   string(data),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd %s: %w", StreamEvents, err)
	}
	log.Printf("[redis] published %s to %s (id=%s)", event.Type, StreamEvents, id)
	return nil
}
