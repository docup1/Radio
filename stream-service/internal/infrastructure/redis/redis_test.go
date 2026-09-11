package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"radio/stream-service/internal/domain/models"
	"radio/stream-service/internal/infrastructure"
)

func newTestStore(t *testing.T) (*QueueStore, *goredis.Client, *miniredis.Miniredis, context.Context) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewQueueStore(rdb), rdb, mr, context.Background()
}

func TestQueueStore_AddListOrderAndPositions(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()

	s1, s2, s3 := uuid.New(), uuid.New(), uuid.New()

	first, err := q.Add(ctx, id, s1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Add(ctx, id, s2); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Add(ctx, id, s3); err != nil {
		t.Fatal(err)
	}

	items, err := q.List(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	for i, want := range []uuid.UUID{s1, s2, s3} {
		if items[i].SongID != want {
			t.Fatalf("items[%d].SongID = %v, want %v", i, items[i].SongID, want)
		}
		if items[i].Position != int64(i) {
			t.Fatalf("items[%d].Position = %d, want %d", i, items[i].Position, i)
		}
		if items[i].ID == uuid.Nil {
			t.Fatal("generated item has nil ID")
		}
	}
	_ = first
	if first.Position != 0 {
		t.Fatalf("first.Position = %d, want 0", first.Position)
	}
}

func TestQueueStore_Remove(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()
	items := make([]*models.QueueItem, 3)
	for i := 0; i < 3; i++ {
		it, _ := q.Add(ctx, id, uuid.New())
		items[i] = it
	}

	if err := q.Remove(ctx, id, items[1].ID); err != nil {
		t.Fatal(err)
	}
	got, _ := q.List(ctx, id)
	if len(got) != 2 || got[0].ID != items[0].ID || got[1].ID != items[2].ID {
		t.Fatalf("after remove: %+v", got)
	}
	if got[1].Position != 1 {
		t.Fatalf("positions not reindexed: %+v", got)
	}

	if err := q.Remove(ctx, id, uuid.New()); err != models.ErrNotFound {
		t.Fatalf("Remove unknown = %v, want ErrNotFound", err)
	}
}

func TestQueueStore_MoveAndClamp(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()
	var ids []uuid.UUID
	var items []*models.QueueItem
	for i := 0; i < 5; i++ {
		it, _ := q.Add(ctx, id, uuid.New())
		ids = append(ids, it.ID)
		items = append(items, it)
	}

	// Move head to the end.
	if err := q.Move(ctx, id, items[0].ID, 10); err != nil {
		t.Fatal(err)
	}
	got, _ := q.List(ctx, id)
	if got[4].ID != items[0].ID {
		t.Fatalf("move-to-end failed: %+v", got)
	}

	// Move tail to the head.
	if err := q.Move(ctx, id, items[4].ID, -5); err != nil {
		t.Fatal(err)
	}
	got, _ = q.List(ctx, id)
	if got[0].ID != items[4].ID {
		t.Fatalf("move-to-head failed: %+v", got)
	}

	if err := q.Move(ctx, id, uuid.New(), 0); err != models.ErrNotFound {
		t.Fatalf("Move unknown = %v, want ErrNotFound", err)
	}
}

func TestQueueStore_CursorActiveLoop(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()

	cursor, err := q.GetCursor(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if cursor != uuid.Nil {
		t.Fatalf("cursor = %v, want nil", cursor)
	}

	itemID := uuid.New()
	if err := q.SetCursor(ctx, id, itemID); err != nil {
		t.Fatal(err)
	}
	got, _ := q.GetCursor(ctx, id)
	if got != itemID {
		t.Fatalf("cursor = %v, want %v", got, itemID)
	}

	if active, _ := q.IsActive(ctx, id); active {
		t.Fatal("stream should be inactive before SetActive")
	}
	if err := q.SetActive(ctx, id); err != nil {
		t.Fatal(err)
	}
	if active, _ := q.IsActive(ctx, id); !active {
		t.Fatal("stream should be active after SetActive")
	}
	ttl := q.rdb.TTL(ctx, "stream:"+id.String()+":active").Val()
	if ttl <= 0 || ttl > ActiveTTL {
		t.Fatalf("active TTL = %v, want <= %v", ttl, ActiveTTL)
	}

	if err := q.SetLoop(ctx, id, true); err != nil {
		t.Fatal(err)
	}
	if v := q.rdb.Get(ctx, "stream:"+id.String()+":loop").Val(); v != "1" {
		t.Fatalf("loop key = %q, want 1", v)
	}
}

func TestQueueStore_Clear(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()
	_, _ = q.Add(ctx, id, uuid.New())
	q.SetActive(ctx, id)
	q.SetCursor(ctx, id, uuid.New())

	if err := q.Clear(ctx, id); err != nil {
		t.Fatal(err)
	}
	if items, _ := q.List(ctx, id); len(items) != 0 {
		t.Fatalf("queue not cleared: %+v", items)
	}
	if active, _ := q.IsActive(ctx, id); active {
		t.Fatal("stream still active after Clear")
	}
}

func TestQueueStore_Status(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()

	it1, _ := q.Add(ctx, id, uuid.New())
	_, _ = q.Add(ctx, id, uuid.New())

	st, err := q.Status(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if st.IsActive || st.QueueLength != 2 {
		t.Fatalf("status before active = %+v", st)
	}

	q.SetActive(ctx, id)
	q.SetCursor(ctx, id, it1.ID)
	st, err = q.Status(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsActive || st.CurrentItemID == nil || *st.CurrentItemID != it1.ID {
		t.Fatalf("status after active = %+v", st)
	}
	if st.Position == nil || *st.Position != 0 {
		t.Fatalf("position = %+v, want 0", st.Position)
	}
}

func TestQueueStore_StatusIdleLeavesCurrentNil(t *testing.T) {
	q, _, _, ctx := newTestStore(t)
	id := uuid.New()
	it, _ := q.Add(ctx, id, uuid.New())
	q.SetCursor(ctx, id, it.ID) // stale cursor without active key
	st, err := q.Status(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if st.IsActive || st.CurrentItemID != nil {
		t.Fatalf("idle status should hide stale cursor: %+v", st)
	}
}

func TestActiveStreamScan(t *testing.T) {
	_, rdb, _, ctx := newTestStore(t)
	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		id := uuid.New()
		ids = append(ids, id)
		key := "stream:" + id.String() + ":active"
		if err := rdb.Set(ctx, key, "1", time.Second).Err(); err != nil {
			t.Fatal(err)
		}
	}
	// Decoy that must not match the scan.
	_ = rdb.Set(ctx, "stream:not-a-uuid:active", "1", time.Second).Err()

	got, err := GetActiveStreamIDs(ctx, rdb)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("scanned %d ids, want 3: %v", len(got), got)
	}
	set := map[uuid.UUID]bool{}
	for _, id := range got {
		set[id] = true
	}
	for _, id := range ids {
		if !set[id] {
			t.Fatalf("missing stream %v in scan", id)
		}
	}
}

func TestActiveStreamsByFreshness(t *testing.T) {
	_, rdb, _, ctx := newTestStore(t)

	oldID := uuid.New()
	freshID := uuid.New()
	rdb.Set(ctx, "stream:"+oldID.String()+":active", "1", 5*time.Second)
	rdb.Set(ctx, "stream:"+freshID.String()+":active", "1", 20*time.Second)

	got, err := GetActiveStreamsByFreshness(ctx, rdb)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got[0] != freshID || got[1] != oldID {
		t.Fatalf("freshness order wrong: %v", got)
	}
}

func TestNewClientPings(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb, err := NewClient(infrastructure.RedisConfig{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

// TestQueueStore_ActiveTTLDefault guards the shared heartbeat contract: the
// sender refreshes the active key every heartbeatInterval (10s), so the TTL
// must comfortably exceed that.
func TestQueueStore_ActiveTTLDefault(t *testing.T) {
	if ActiveTTL != 30*time.Second {
		t.Fatalf("ActiveTTL = %v, want 30s", ActiveTTL)
	}
}
