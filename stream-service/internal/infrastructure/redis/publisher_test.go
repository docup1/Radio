package redis

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func TestPublisher_WritesStreamEvents(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	p := NewPublisher(rdb)
	ctx := context.Background()
	streamID := uuid.New()
	itemID := uuid.New()
	songID := uuid.New()

	if err := p.PublishStreamStarted(ctx, streamID, itemID, songID); err != nil {
		t.Fatal(err)
	}
	if err := p.PublishSongChanged(ctx, streamID, itemID, songID); err != nil {
		t.Fatal(err)
	}
	if err := p.PublishStreamStopped(ctx, streamID); err != nil {
		t.Fatal(err)
	}

	key := "stream:" + streamID.String() + ":events"
	msgs, err := rdb.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("events = %d, want 3", len(msgs))
	}
	if len(msgs[0].Values) != 1 {
		t.Fatalf("event entry %+v", msgs[0].Values)
	}
	data, ok := msgs[0].Values["data"].(string)
	if !ok {
		t.Fatalf("data field missing: %+v", msgs[0].Values)
	}
	var ev struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		t.Fatalf("bad event json: %v", err)
	}
	if ev.Type != "stream_started" {
		t.Fatalf("event type = %q, want stream_started", ev.Type)
	}
	if !strings.Contains(data, itemID.String()) || !strings.Contains(data, songID.String()) {
		t.Fatalf("payload missing ids: %s", data)
	}
}

func TestPublisher_StopEventHasNoPayload(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	p := NewPublisher(rdb)
	streamID := uuid.New()
	if err := p.PublishStreamStopped(context.Background(), streamID); err != nil {
		t.Fatal(err)
	}
	key := "stream:" + streamID.String() + ":events"
	msgs, _ := rdb.XRange(context.Background(), key, "-", "+").Result()
	data := msgs[0].Values["data"].(string)
	if strings.Contains(data, "payload") && !strings.Contains(data, `"payload":null`) {
		t.Fatalf("stop event should not carry a payload: %s", data)
	}
}

func TestEventsKeySharedWithSender(t *testing.T) {
	// The sender reads stream:{id}:events via its own consumer; the key must
	// match the exact pattern both sides use.
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	p := NewPublisher(rdb)
	streamID := uuid.New()
	if err := p.PublishQueueUpdated(context.Background(), streamID); err != nil {
		t.Fatal(err)
	}
	wantKey := "stream:" + streamID.String() + ":events"
	got, _ := rdb.Keys(context.Background(), "stream:*:events").Result()
	if len(got) != 1 || got[0] != wantKey {
		t.Fatalf("keys = %v, want [%s]", got, wantKey)
	}
}
