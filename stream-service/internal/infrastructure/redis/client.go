package redis

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

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

// GetActiveStreamsByFreshness scans active stream keys and orders them by
// heartbeat freshness (most recently refreshed first). The active key carries
// a TTL that is reset on every heartbeat, so a larger remaining TTL means a
// fresher stream.
func GetActiveStreamsByFreshness(ctx context.Context, rdb *redis.Client) ([]uuid.UUID, error) {
	ids, err := GetActiveStreamIDs(ctx, rdb)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return ids, nil
	}

	pipe := rdb.Pipeline()
	ttlCmds := make(map[string]*redis.DurationCmd, len(ids))
	for _, id := range ids {
		key := fmt.Sprintf("stream:%s:active", id)
		ttlCmds[key] = pipe.TTL(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("read active TTLs: %w", err)
	}

	fresh := make([]uuid.UUID, len(ids))
	copy(fresh, ids)
	ttlOf := func(id uuid.UUID) time.Duration {
		key := fmt.Sprintf("stream:%s:active", id)
		ttl, _ := ttlCmds[key].Result()
		return ttl
	}
	sort.SliceStable(fresh, func(i, j int) bool {
		return ttlOf(fresh[i]) > ttlOf(fresh[j])
	})
	return fresh, nil
}
