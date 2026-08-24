package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"radio/sender-service/internal/infrastructure"
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
