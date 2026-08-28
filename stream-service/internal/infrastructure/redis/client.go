package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"radio/stream-service/internal/infrastructure"
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
	return rdb, nil
}

// GetActiveStreamIDs scans Redis for stream:*:active keys and returns the stream UUIDs.
func GetActiveStreamIDs(ctx context.Context, rdb *redis.Client) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	iter := rdb.Scan(ctx, 0, "stream:*:active", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		parts := strings.SplitN(key, ":", 3)
		if len(parts) != 3 {
			continue
		}
		id, err := uuid.Parse(parts[1])
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("scan active streams: %w", err)
	}
	return ids, nil
}
