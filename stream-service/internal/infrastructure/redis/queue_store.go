package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"radio/stream-service/internal/domain/models"
)

// QueueStore manages the ephemeral playback queue in Redis.
//
// Keys (per stream):
//   - stream:{id}:queue  LIST of JSON {"item_id","song_id"} — index == position
//   - stream:{id}:cursor STRING item_id — currently playing entry
//   - stream:{id}:active STRING "1" with TTL — liveness marker (refreshed by sender heartbeat)
//   - stream:{id}:loop   STRING "1"/"0" — loop flag snapshot for the active session
type QueueStore struct {
	rdb *goredis.Client
}

func NewQueueStore(rdb *goredis.Client) *QueueStore {
	return &QueueStore{rdb: rdb}
}

const ActiveTTL = 30 * time.Second

type queueEntry struct {
	ItemID string `json:"item_id"`
	SongID string `json:"song_id"`
}

func queueKey(id uuid.UUID) string      { return fmt.Sprintf("stream:%s:queue", id) }
func cursorKey(id uuid.UUID) string     { return fmt.Sprintf("stream:%s:cursor", id) }
func activeKey(id uuid.UUID) string     { return fmt.Sprintf("stream:%s:active", id) }
func loopKey(id uuid.UUID) string       { return fmt.Sprintf("stream:%s:loop", id) }

// Add appends a song to the tail of the queue.
func (s *QueueStore) Add(ctx context.Context, streamID, songID uuid.UUID) (*models.QueueItem, error) {
	key := queueKey(streamID)
	pos, err := s.rdb.LLen(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("llen %s: %w", key, err)
	}
	item := &models.QueueItem{
		ID:       uuid.New(),
		SongID:   songID,
		Position: pos,
	}
	data, err := json.Marshal(queueEntry{ItemID: item.ID.String(), SongID: songID.String()})
	if err != nil {
		return nil, err
	}
	if err := s.rdb.RPush(ctx, key, string(data)).Err(); err != nil {
		return nil, fmt.Errorf("rpush %s: %w", key, err)
	}
	return item, nil
}

// List returns all queue items with their positions.
func (s *QueueStore) List(ctx context.Context, streamID uuid.UUID) ([]*models.QueueItem, error) {
	entries, err := s.snapshot(ctx, streamID)
	if err != nil {
		return nil, err
	}
	items := make([]*models.QueueItem, len(entries))
	for i, e := range entries {
		items[i] = &models.QueueItem{ID: e.ItemID, SongID: e.SongID, Position: int64(i)}
	}
	return items, nil
}

// Remove deletes an item from the queue by ID.
func (s *QueueStore) Remove(ctx context.Context, streamID, itemID uuid.UUID) error {
	entries, err := s.snapshot(ctx, streamID)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	found := false
	for _, e := range entries {
		if e.ItemID == itemID {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return models.ErrNotFound
	}
	return s.rewrite(ctx, streamID, filtered)
}

// Move reorders an item to a new position.
func (s *QueueStore) Move(ctx context.Context, streamID, itemID uuid.UUID, newPosition int64) error {
	entries, err := s.snapshot(ctx, streamID)
	if err != nil {
		return err
	}
	idx := -1
	for i, e := range entries {
		if e.ItemID == itemID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return models.ErrNotFound
	}
	e := entries[idx]
	entries = append(entries[:idx], entries[idx+1:]...)
	pos := newPosition
	if pos < 0 {
		pos = 0
	}
	if pos > int64(len(entries)) {
		pos = int64(len(entries))
	}
	entries = append(entries[:pos], append([]entry{e}, entries[pos:]...)...)
	return s.rewrite(ctx, streamID, entries)
}

type entry struct {
	ItemID uuid.UUID
	SongID uuid.UUID
}

func (s *QueueStore) snapshot(ctx context.Context, streamID uuid.UUID) ([]entry, error) {
	raw, err := s.rdb.LRange(ctx, queueKey(streamID), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange %s: %w", queueKey(streamID), err)
	}
	entries := make([]entry, 0, len(raw))
	for _, r := range raw {
		var qe queueEntry
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
		entries = append(entries, entry{ItemID: itemID, SongID: songID})
	}
	return entries, nil
}

func (s *QueueStore) rewrite(ctx context.Context, streamID uuid.UUID, entries []entry) error {
	key := queueKey(streamID)
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, key)
	for _, e := range entries {
		data, _ := json.Marshal(queueEntry{ItemID: e.ItemID.String(), SongID: e.SongID.String()})
		pipe.RPush(ctx, key, string(data))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("rewrite %s: %w", key, err)
	}
	return nil
}

// GetCursor returns the current item ID or uuid.Nil if not set.
func (s *QueueStore) GetCursor(ctx context.Context, streamID uuid.UUID) (uuid.UUID, error) {
	val, err := s.rdb.Get(ctx, cursorKey(streamID)).Result()
	if err != nil {
		if err == goredis.Nil {
			return uuid.Nil, nil
		}
		return uuid.Nil, fmt.Errorf("get cursor %s: %w", streamID, err)
	}
	return uuid.Parse(val)
}

// SetCursor stores the currently playing item ID.
func (s *QueueStore) SetCursor(ctx context.Context, streamID, itemID uuid.UUID) error {
	return s.rdb.Set(ctx, cursorKey(streamID), itemID.String(), 0).Err()
}

// SetActive marks the stream as live with a TTL. The sender refreshes it via heartbeat.
func (s *QueueStore) SetActive(ctx context.Context, streamID uuid.UUID) error {
	return s.rdb.Set(ctx, activeKey(streamID), "1", ActiveTTL).Err()
}

// IsActive reports whether the stream is currently live.
func (s *QueueStore) IsActive(ctx context.Context, streamID uuid.UUID) (bool, error) {
	n, err := s.rdb.Exists(ctx, activeKey(streamID)).Result()
	if err != nil {
		return false, fmt.Errorf("exists %s: %w", streamID, err)
	}
	return n > 0, nil
}

// SetLoop stores the loop flag for the active session (read by sender on EOF).
func (s *QueueStore) SetLoop(ctx context.Context, streamID uuid.UUID, loop bool) error {
	v := "0"
	if loop {
		v = "1"
	}
	return s.rdb.Set(ctx, loopKey(streamID), v, 0).Err()
}

// Clear wipes all runtime keys for the stream (queue, cursor, active, loop).
func (s *QueueStore) Clear(ctx context.Context, streamID uuid.UUID) error {
	return s.rdb.Del(ctx,
		queueKey(streamID), cursorKey(streamID), activeKey(streamID), loopKey(streamID),
	).Err()
}

// Status builds the full runtime status of a stream.
func (s *QueueStore) Status(ctx context.Context, streamID uuid.UUID) (*models.StreamStatus, error) {
	active, err := s.IsActive(ctx, streamID)
	if err != nil {
		return nil, err
	}
	items, err := s.List(ctx, streamID)
	if err != nil {
		return nil, err
	}
	st := &models.StreamStatus{
		IsActive:    active,
		QueueLength: int64(len(items)),
	}
	if !active {
		return st, nil
	}
	cursor, err := s.GetCursor(ctx, streamID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.ID == cursor {
			id := it.ID
			song := it.SongID
			pos := it.Position
			st.CurrentItemID = &id
			st.CurrentSongID = &song
			st.Position = &pos
			break
		}
	}
	return st, nil
}
