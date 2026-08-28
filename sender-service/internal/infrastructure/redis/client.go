package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	consumerGroup   = "sender"
	consumerName    = "sender-1"
	activeTTL       = 30 * time.Second
)

// QueueEntry is one element of the runtime queue (owned by stream-service).
type QueueEntry struct {
	ItemID uuid.UUID
	SongID uuid.UUID
}

type Client struct {
	rdb *redis.Client
	mu  sync.Mutex
}

func NewClient(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func queueKey(id uuid.UUID) string   { return fmt.Sprintf("stream:%s:queue", id) }
func cursorKey(id uuid.UUID) string  { return fmt.Sprintf("stream:%s:cursor", id) }
func activeKey(id uuid.UUID) string  { return fmt.Sprintf("stream:%s:active", id) }
func loopKey(id uuid.UUID) string    { return fmt.Sprintf("stream:%s:loop", id) }
func eventsKey(id uuid.UUID) string  { return fmt.Sprintf("stream:%s:events", id) }

// --- Consumer Group ---

func (c *Client) EnsureConsumerGroup(ctx context.Context, streamID uuid.UUID) error {
	key := eventsKey(streamID)
	err := c.rdb.XGroupCreateMkStream(ctx, key, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("xgroup create %s: %w", key, err)
	}
	return nil
}

type EventMessage struct {
	ID      string
	Type    string
	Payload json.RawMessage
}

func (c *Client) ReadEvents(ctx context.Context, streamID uuid.UUID) ([]*EventMessage, error) {
	key := eventsKey(streamID)

	res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{key, ">"},
		Count:    10,
		Block:    1 * time.Second,
	}).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil {
			return nil, nil
		}
		if isNogroupErr(err) {
			log.Printf("[redis] NOGROUP for %s, recreating consumer group", key)
			_ = c.rdb.XGroupDestroy(ctx, key, consumerGroup).Err()
			_ = c.EnsureConsumerGroup(ctx, streamID)
			return nil, nil
		}
		return nil, fmt.Errorf("xreadgroup %s: %w", key, err)
	}

	return c.parseEvents(res), nil
}

func (c *Client) AckEvent(ctx context.Context, streamID uuid.UUID, eventID string) error {
	return c.rdb.XAck(ctx, eventsKey(streamID), consumerGroup, eventID).Err()
}

// ReadPendingEvents recovers unacknowledged messages (PEL) on startup.
func (c *Client) ReadPendingEvents(ctx context.Context, streamID uuid.UUID) ([]*EventMessage, error) {
	key := eventsKey(streamID)

	res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{key, "0"},
		Count:    10,
	}).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil {
			return nil, nil
		}
		if isNogroupErr(err) {
			_ = c.EnsureConsumerGroup(ctx, streamID)
			return nil, nil
		}
		return nil, fmt.Errorf("xreadgroup pending %s: %w", key, err)
	}

	return c.parseEvents(res), nil
}

func (c *Client) parseEvents(res []redis.XStream) []*EventMessage {
	var events []*EventMessage
	for _, s := range res {
		for _, msg := range s.Messages {
			raw, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}
			var parsed struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload,omitempty"`
			}
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				continue
			}
			events = append(events, &EventMessage{
				ID:      msg.ID,
				Type:    parsed.Type,
				Payload: parsed.Payload,
			})
		}
	}
	return events
}

func isNogroupErr(err error) bool {
	return err != nil && len(err.Error()) >= 7 && err.Error()[:7] == "NOGROUP"
}

// PublishEvent appends an event to the stream's event log (used for stream_ended).
func (c *Client) PublishEvent(ctx context.Context, streamID uuid.UUID, eventType string, payload interface{}) error {
	data, err := json.Marshal(struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload,omitempty"`
	}{Type: eventType, Payload: payload})
	if err != nil {
		return err
	}
	return c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: eventsKey(streamID),
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{"data": string(data)},
	}).Err()
}

// --- Queue / cursor / runtime state (written by stream-service, read here) ---

type queueEntryDTO struct {
	ItemID string `json:"item_id"`
	SongID string `json:"song_id"`
}

// SnapshotQueue reads the full queue. Empty slice when no queue exists.
func (c *Client) SnapshotQueue(ctx context.Context, streamID uuid.UUID) ([]QueueEntry, error) {
	raw, err := c.rdb.LRange(ctx, queueKey(streamID), 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("lrange queue %s: %w", streamID, err)
	}
	entries := make([]QueueEntry, 0, len(raw))
	for _, r := range raw {
		var qe queueEntryDTO
		if err := json.Unmarshal([]byte(r), &qe); err != nil {
			continue
		}
		itemID, err := uuid.Parse(qe.ItemID)
		if err != nil {
			continue
		}
		songID, err := uuid.Parse(qe.SongID)
		if err != nil {
			continue
		}
		entries = append(entries, QueueEntry{ItemID: itemID, SongID: songID})
	}
	return entries, nil
}

// GetCursor returns the current item ID or uuid.Nil.
func (c *Client) GetCursor(ctx context.Context, streamID uuid.UUID) (uuid.UUID, error) {
	val, err := c.rdb.Get(ctx, cursorKey(streamID)).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("get cursor %s: %w", streamID, err)
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse cursor %s: %w", val, err)
	}
	return id, nil
}

// SetCursor stores the currently playing item ID.
func (c *Client) SetCursor(ctx context.Context, streamID, itemID uuid.UUID) error {
	return c.rdb.Set(ctx, cursorKey(streamID), itemID.String(), 0).Err()
}

// GetLoop returns the loop flag of the active session.
func (c *Client) GetLoop(ctx context.Context, streamID uuid.UUID) (bool, error) {
	val, err := c.rdb.Get(ctx, loopKey(streamID)).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("get loop %s: %w", streamID, err)
	}
	return val == "1", nil
}

// RefreshActive extends the active key TTL (heartbeat). No-op if key expired.
func (c *Client) RefreshActive(ctx context.Context, streamID uuid.UUID) error {
	return c.rdb.Expire(ctx, activeKey(streamID), activeTTL).Err()
}

// IsActive reports whether the stream is live.
func (c *Client) IsActive(ctx context.Context, streamID uuid.UUID) (bool, error) {
	n, err := c.rdb.Exists(ctx, activeKey(streamID)).Result()
	if err != nil {
		return false, fmt.Errorf("exists %s: %w", streamID, err)
	}
	return n > 0, nil
}

// DeleteStreamState wipes all runtime keys of the stream (queue, cursor, active, loop, events).
func (c *Client) DeleteStreamState(ctx context.Context, streamID uuid.UUID) error {
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, queueKey(streamID), cursorKey(streamID), activeKey(streamID), loopKey(streamID))
	pipe.Del(ctx, eventsKey(streamID))
	_, err := pipe.Exec(ctx)
	return err
}

// GetActiveStreams returns all currently live stream IDs.
func (c *Client) GetActiveStreams(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	iter := c.rdb.Scan(ctx, 0, "stream:*:active", 100).Iterator()
	for iter.Next(ctx) {
		parts := strings.SplitN(iter.Val(), ":", 3)
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
