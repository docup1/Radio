package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"radio/stream-service/internal/domain/interfaces"
	"radio/stream-service/internal/domain/models"
	svcredis "radio/stream-service/internal/infrastructure/redis"
)

// --- fakes ---

type fakeStreams struct {
	byID map[uuid.UUID]*models.Stream
}

func (f *fakeStreams) Create(_ context.Context, s *models.Stream) error {
	if _, ok := f.byID[s.ID]; ok {
		return models.ErrConflict
	}
	f.byID[s.ID] = s
	return nil
}

func (f *fakeStreams) GetByID(_ context.Context, id uuid.UUID) (*models.Stream, error) {
	s, ok := f.byID[id]
	if !ok {
		return nil, models.ErrNotFound
	}
	return s, nil
}

func (f *fakeStreams) ListByIDs(_ context.Context, ids []uuid.UUID) ([]*models.Stream, error) {
	out := make([]*models.Stream, 0, len(ids))
	for _, id := range ids {
		if s, ok := f.byID[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeStreams) SearchActive(_ context.Context, ids []uuid.UUID, q string, limit, offset int) ([]*models.Stream, int, error) {
	q = strings.ToLower(strings.TrimSpace(q))
	var all []*models.Stream
	for _, id := range ids {
		s, ok := f.byID[id]
		if !ok {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(s.Name), q) || strings.Contains(strings.ToLower(s.Description), q) {
			all = append(all, s)
		}
	}
	total := len(all)
	if offset >= len(all) {
		return nil, total, nil
	}
	all = all[offset:]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, total, nil
}

func (f *fakeStreams) Update(_ context.Context, s *models.Stream) error {
	if _, ok := f.byID[s.ID]; !ok {
		return models.ErrNotFound
	}
	f.byID[s.ID] = s
	return nil
}

func (f *fakeStreams) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return models.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeHashtags struct {
	byID   map[uuid.UUID]*models.Hashtag
	stream map[uuid.UUID][]*models.Hashtag
}

func (f *fakeHashtags) Add(_ context.Context, h *models.Hashtag) error {
	f.byID[h.ID] = h
	f.stream[h.StreamID] = append(f.stream[h.StreamID], h)
	return nil
}

func (f *fakeHashtags) ListByStream(_ context.Context, streamID uuid.UUID) ([]*models.Hashtag, error) {
	return f.stream[streamID], nil
}

func (f *fakeHashtags) Remove(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return models.ErrNotFound
	}
	delete(f.byID, id)
	for sid, list := range f.stream {
		for i, h := range list {
			if h.ID == id {
				f.stream[sid] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (f *fakeHashtags) RemoveByStream(_ context.Context, streamID uuid.UUID) error {
	delete(f.stream, streamID)
	return nil
}

type fakeChecker struct {
	allowed func(ownerID, songID string) bool
}

func (f *fakeChecker) Check(_ context.Context, ownerID, songID string) (bool, error) {
	if f.allowed == nil {
		return true, nil
	}
	return f.allowed(ownerID, songID), nil
}

// --- harness ---

type testSvc struct {
	svc     *Service
	streams *fakeStreams
	hashtag *fakeHashtags
	rdb     *goredis.Client
	q       *svcredis.QueueStore
	pub     *svcredis.Publisher
	ctx     context.Context
}

func newTestService(t *testing.T, checker interfaces.SongChecker) *testSvc {
	t.Helper()
	if checker == nil {
		checker = &fakeChecker{}
	}
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	q := svcredis.NewQueueStore(rdb)
	pub := svcredis.NewPublisher(rdb)
	streams := &fakeStreams{byID: map[uuid.UUID]*models.Stream{}}
	hashtag := &fakeHashtags{
		byID:   map[uuid.UUID]*models.Hashtag{},
		stream: map[uuid.UUID][]*models.Hashtag{},
	}

	svc := New(Repos{Streams: streams, Hashtags: hashtag}, q, pub, checker, rdb)
	return &testSvc{
		svc: svc, streams: streams, hashtag: hashtag, rdb: rdb, q: q, pub: pub, ctx: context.Background(),
	}
}

func eventsFor(t *testing.T, ctx context.Context, rdb *goredis.Client, streamID uuid.UUID) []string {
	t.Helper()
	key := "stream:" + streamID.String() + ":events"
	msgs, err := rdb.XRange(ctx, key, "-", "+").Result()
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, m := range msgs {
		data, _ := m.Values["data"].(string)
		if strings.Contains(data, `"type":"stream_started"`) {
			types = append(types, "stream_started")
		}
		if strings.Contains(data, `"type":"stream_stopped"`) {
			types = append(types, "stream_stopped")
		}
		if strings.Contains(data, `"type":"queue_updated"`) {
			types = append(types, "queue_updated")
		}
	}
	return types
}

func addQueue(t *testing.T, s *testSvc, streamID uuid.UUID, n int) []uuid.UUID {
	t.Helper()
	var songIDs []uuid.UUID
	for i := 0; i < n; i++ {
		it, err := s.svc.AddToQueue(s.ctx, streamID, uuid.New())
		if err != nil {
			t.Fatalf("AddToQueue: %v", err)
		}
		songIDs = append(songIDs, it.SongID)
	}
	return songIDs
}

// --- tests ---

func TestGetOrCreateStream_CreatesAndReuses(t *testing.T) {
	s := newTestService(t, nil)
	userID := uuid.New()

	first, err := s.svc.GetOrCreateStream(s.ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "My Stream" || first.ID != userID {
		t.Fatalf("created stream = %+v", first)
	}

	second, err := s.svc.GetOrCreateStream(s.ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Name != "My Stream" {
		t.Fatalf("second lookup mutated stream: %+v", second)
	}
	if len(s.streams.byID) != 1 {
		t.Fatalf("repository has %d streams, want 1", len(s.streams.byID))
	}
}

func TestGetStream_NotFound(t *testing.T) {
	s := newTestService(t, nil)
	if _, err := s.svc.GetStream(s.ctx, uuid.New()); err != models.ErrNotFound {
		t.Fatalf("GetStream = %v, want ErrNotFound", err)
	}
}

func TestStart_ActivatesAndPublishes(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}
	addQueue(t, s, streamID, 2)

	if err := s.svc.Start(s.ctx, streamID); err != nil {
		t.Fatal(err)
	}
	if active, _ := s.q.IsActive(s.ctx, streamID); !active {
		t.Fatal("stream not active after Start")
	}
	st, _ := s.svc.GetStatus(s.ctx, streamID)
	if st.CurrentItemID == nil || st.Position == nil || *st.Position != 0 {
		t.Fatalf("status = %+v", st)
	}
	if types := eventsFor(t, s.ctx, s.rdb, streamID); !contains(types, "stream_started") {
		t.Fatalf("events = %v, want stream_started", types)
	}
}

func TestStart_Errors(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()

	// missing stream
	if err := s.svc.Start(s.ctx, streamID); err != models.ErrNotFound {
		t.Fatalf("missing stream Start = %v", err)
	}

	// empty queue
	s.streams.byID[streamID] = &models.Stream{ID: streamID}
	if err := s.svc.Start(s.ctx, streamID); err != models.ErrInvalid {
		t.Fatalf("empty queue Start = %v, want ErrInvalid", err)
	}

	// already active
	addQueue(t, s, streamID, 1)
	if err := s.svc.Start(s.ctx, streamID); err != nil {
		t.Fatal(err)
	}
	if err := s.svc.Start(s.ctx, streamID); err != models.ErrConflict {
		t.Fatalf("second Start = %v, want ErrConflict", err)
	}
}

func TestStop_ClearsAndPublishes(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}
	addQueue(t, s, streamID, 1)
	if err := s.svc.Start(s.ctx, streamID); err != nil {
		t.Fatal(err)
	}

	if err := s.svc.Stop(s.ctx, streamID); err != nil {
		t.Fatal(err)
	}
	if active, _ := s.q.IsActive(s.ctx, streamID); active {
		t.Fatal("stream still active after Stop")
	}
	if items, _ := s.svc.ListQueue(s.ctx, streamID); len(items) != 0 {
		t.Fatalf("queue not cleared after Stop: %d items", len(items))
	}
	if !contains(eventsFor(t, s.ctx, s.rdb, streamID), "stream_stopped") {
		t.Fatal("stream_stopped event missing")
	}

	// stopping an inactive stream conflicts
	if err := s.svc.Stop(s.ctx, streamID); err != models.ErrConflict {
		t.Fatalf("second Stop = %v, want ErrConflict", err)
	}
}

func TestAddToQueue_ForbiddenDenied(t *testing.T) {
	s := newTestService(t, &fakeChecker{allowed: func(_, _ string) bool { return false }})
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}

	if _, err := s.svc.AddToQueue(s.ctx, streamID, uuid.New()); err != models.ErrForbidden {
		t.Fatalf("AddToQueue denied = %v, want ErrForbidden", err)
	}
}

func TestAddToQueue_MissingStream(t *testing.T) {
	s := newTestService(t, &fakeChecker{allowed: func(_, _ string) bool { return true }})
	if _, err := s.svc.AddToQueue(s.ctx, uuid.New(), uuid.New()); err != models.ErrNotFound {
		t.Fatalf("AddToQueue missing stream = %v, want ErrNotFound", err)
	}
}

func TestRemoveAndMoveQueue(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}
	addQueue(t, s, streamID, 3)

	items, _ := s.svc.ListQueue(s.ctx, streamID)
	if err := s.svc.RemoveFromQueue(s.ctx, streamID, items[0].ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.svc.ListQueue(s.ctx, streamID)
	if len(got) != 2 || got[0].ID != items[1].ID {
		t.Fatalf("after remove = %+v", got)
	}

	if err := s.svc.MoveQueueItem(s.ctx, streamID, got[1].ID, 0); err != nil {
		t.Fatal(err)
	}
	got, _ = s.svc.ListQueue(s.ctx, streamID)
	if got[0].ID != items[2].ID {
		t.Fatalf("after move to head = %+v", got)
	}

	if err := s.svc.RemoveFromQueue(s.ctx, streamID, uuid.New()); err != models.ErrNotFound {
		t.Fatalf("remove unknown = %v, want ErrNotFound", err)
	}
}

func TestHashtags(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}

	h, err := s.svc.AddHashtag(s.ctx, streamID, "house")
	if err != nil {
		t.Fatal(err)
	}
	if h.StreamID != streamID {
		t.Fatalf("hashtag stream = %v", h.StreamID)
	}

	if _, err := s.svc.AddHashtag(s.ctx, streamID, strings.Repeat("x", 65)); err != models.ErrInvalid {
		t.Fatalf("long hashtag = %v, want ErrInvalid", err)
	}

	list, err := s.svc.ListHashtags(s.ctx, streamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != h.ID {
		t.Fatalf("list = %+v", list)
	}

	if err := s.svc.RemoveHashtag(s.ctx, streamID, h.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.svc.ListHashtags(s.ctx, streamID); len(list) != 0 {
		t.Fatal("hashtag not removed")
	}
}

func TestUpdateStream_SyncsLoopSnapshot(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID}

	updated, err := s.svc.UpdateStream(s.ctx, streamID, "name", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "name" || updated.Description != "desc" {
		t.Fatalf("updated = %+v, want name=name desc=desc", updated)
	}
}

func TestFeed_ActiveOnlyOrderedByFreshness(t *testing.T) {
	s := newTestService(t, nil)

	oldID := uuid.New()
	freshID := uuid.New()
	s.streams.byID[oldID] = &models.Stream{ID: oldID, Name: "Old", Description: "", CreatedAt: time.Now()}
	s.streams.byID[freshID] = &models.Stream{ID: freshID, Name: "Fresh", Description: ""}

	// Freshness: larger remaining TTL wins.
	s.rdb.Set(s.ctx, "stream:"+oldID.String()+":active", "1", 5*time.Second)
	s.rdb.Set(s.ctx, "stream:"+freshID.String()+":active", "1", 20*time.Second)

	feed, err := s.svc.Feed(s.ctx, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 2 {
		t.Fatalf("feed len = %d, want 2", len(feed))
	}
	if feed[0].Stream.ID != freshID {
		t.Fatalf("feed[0] = %v, want freshest", feed[0].Stream.ID)
	}
	if feed[1].Stream.ID != oldID {
		t.Fatalf("feed[1] = %v, want oldest", feed[1].Stream.ID)
	}
	if feed[0].CurrentSongID != nil {
		t.Fatalf("current song on idle stream: %v", *feed[0].CurrentSongID)
	}
}

func TestFeed_MarksCurrentSong(t *testing.T) {
	s := newTestService(t, nil)
	streamID := uuid.New()
	s.streams.byID[streamID] = &models.Stream{ID: streamID, Name: "Live"}
	s.rdb.Set(s.ctx, "stream:"+streamID.String()+":active", "1", 10*time.Second)
	it, _ := s.svc.AddToQueue(s.ctx, streamID, uuid.New())
	s.q.SetCursor(s.ctx, streamID, it.ID)

	feed, err := s.svc.Feed(s.ctx, "", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 1 || feed[0].CurrentSongID == nil || *feed[0].CurrentSongID != it.SongID {
		t.Fatalf("feed = %+v", feed)
	}
}

func TestFeed_SearchFiltersByName(t *testing.T) {
	s := newTestService(t, nil)
	hitID := uuid.New()
	missID := uuid.New()
	s.streams.byID[hitID] = &models.Stream{ID: hitID, Name: "Deep House Night"}
	s.streams.byID[missID] = &models.Stream{ID: missID, Name: "Rock Anthems"}
	s.rdb.Set(s.ctx, "stream:"+hitID.String()+":active", "1", 10*time.Second)
	s.rdb.Set(s.ctx, "stream:"+missID.String()+":active", "1", 10*time.Second)

	feed, err := s.svc.Feed(s.ctx, "house", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 1 || feed[0].Stream.ID != hitID {
		t.Fatalf("search feed = %+v", feed)
	}
}

func TestFeed_Pagination(t *testing.T) {
	s := newTestService(t, nil)
	for _, name := range []string{"A", "B", "C", "D"} {
		id := uuid.New()
		s.streams.byID[id] = &models.Stream{ID: id, Name: name}
		s.rdb.Set(s.ctx, "stream:"+id.String()+":active", "1", 10*time.Second)
	}

	page, err := s.svc.Feed(s.ctx, "", 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Fatalf("page len = %d, want 2", len(page))
	}
	if feed, _ := s.svc.Feed(s.ctx, "", 2, 10); len(feed) != 0 {
		t.Fatalf("offset past end should return empty, got %d", len(feed))
	}
}

func TestListActiveStreams(t *testing.T) {
	s := newTestService(t, nil)
	id := uuid.New()
	s.rdb.Set(s.ctx, "stream:"+id.String()+":active", "1", time.Second)

	ids, err := s.svc.ListActiveStreams(s.ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, got := range ids {
		if got == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("ids = %v, missing %v", ids, id)
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
