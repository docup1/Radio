package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	contenthttp "radio/sender-service/internal/infrastructure/http"
	senderredis "radio/sender-service/internal/infrastructure/redis"
)

const testChunkSize = 4096

// testFrame returns one valid MPEG1 Layer III frame (128kbps/44.1kHz, 417 B).
func testFrame() []byte {
	f := make([]byte, 417)
	f[0] = 0xFF
	f[1] = 0xFB
	f[2] = 0x90 // bitrate index 9 (128k), sample rate index 0 (44.1k)
	f[3] = 0x00
	for i := 4; i < len(f); i++ {
		f[i] = 0xCC
	}
	return f
}

func testAudio(frames int) []byte {
	out := make([]byte, 0, frames*417)
	for i := 0; i < frames; i++ {
		out = append(out, testFrame()...)
	}
	return out
}

// fakeContent serves audio chunks exactly like the content-service HTTP audio
// endpoint: honor Range, emit Content-Range totals, 416 past EOF.
type fakeContent struct {
	files     map[string][]byte
	chunkSize int
}

func (fc *fakeContent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	songID := parts[len(parts)-2] // .../songs/<id>/audio
	audio, ok := fc.files[songID]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	offset := 0
	if rg := r.Header.Get("Range"); rg != "" {
		var start int64
		_, _ = fmt.Sscanf(rg, "bytes=%d-", &start)
		offset = int(start)
	}
	if offset >= len(audio) {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", len(audio)))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}
	end := offset + fc.chunkSize - 1
	if end >= len(audio) {
		end = len(audio) - 1
	}
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", offset, end, len(audio)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(http.StatusPartialContent)
	_, _ = w.Write(audio[offset : end+1])
}

type testEnv struct {
	svc      *Service
	hub      *Hub
	sredis   *senderredis.Client
	raw      *goredis.Client
	streamID uuid.UUID
	mr       *miniredis.Miniredis
	server   *httptest.Server
}

// newTestEnv builds a Service backed by miniredis and an in-process fake
// content server. All calls, including Start, happen lazily in the tests.
func newTestEnv(t *testing.T, files map[string][]byte) *testEnv {
	t.Helper()

	mr := miniredis.RunT(t)
	raw := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = raw.Close() })

	sredis, err := senderredis.NewClient(mr.Addr())
	if err != nil {
		t.Fatalf("sender redis client: %v", err)
	}
	t.Cleanup(func() { _ = sredis.Close() })

	server := httptest.NewServer(&fakeContent{files: files, chunkSize: testChunkSize})
	t.Cleanup(server.Close)

	hub := NewHub()
	svc := NewService(
		contenthttp.NewContentClient(server.URL, testChunkSize),
		sredis,
		hub,
		SenderConfig{
			ContentServiceURL: server.URL,
			ChunkSize:         testChunkSize,
			Bitrate:           128000,
			BufferSeconds:     5,
			PrefetchCount:     8,
			NextSongPrefetch:  1,
		},
	)
	return &testEnv{
		svc:      svc,
		hub:      hub,
		sredis:   sredis,
		raw:      raw,
		streamID: uuid.New(),
		mr:       mr,
		server:   server,
	}
}

// newTestEnvChunkSize builds an env whose sender delivers frame-aligned chunks
// of roughly the given byte size (content client + fake server and sender
// config all share it). The standard env uses testChunkSize (4096 B), whose
// 261 ms chunks are smaller than the 800 ms pacing lead, so pacing never
// engages there. Larger chunk sizes are needed to observe delivery pacing.
func newTestEnvChunkSize(t *testing.T, file []byte, chunkSize int64) *testEnv {
	t.Helper()

	fake := &fakeContent{files: map[string][]byte{songID(1).String(): file}, chunkSize: int(chunkSize)}
	mr := miniredis.RunT(t)
	raw := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = raw.Close() })

	sredis, err := senderredis.NewClient(mr.Addr())
	if err != nil {
		t.Fatalf("sender redis client: %v", err)
	}
	t.Cleanup(func() { _ = sredis.Close() })

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	hub := NewHub()
	svc := NewService(
		contenthttp.NewContentClient(server.URL, chunkSize),
		sredis,
		hub,
		SenderConfig{
			ContentServiceURL: server.URL,
			ChunkSize:         chunkSize,
			Bitrate:           128000,
			BufferSeconds:     5,
			PrefetchCount:     8,
			NextSongPrefetch:  1,
		},
	)
	return &testEnv{
		svc:      svc,
		hub:      hub,
		sredis:   sredis,
		raw:      raw,
		streamID: uuid.New(),
		mr:       mr,
		server:   server,
	}
}

// seedQueue writes queue entries into miniredis using the exact key format and
// payload shape the sender's SnapshotQueue expects. RPush preserves order.
func (e *testEnv) seedQueue(t *testing.T, entries ...queueItem) {
	t.Helper()
	key := fmt.Sprintf("stream:%s:queue", e.streamID)
	for _, it := range entries {
		row, _ := json.Marshal(map[string]string{
			"item_id": it.ItemID.String(),
			"song_id": it.SongID.String(),
		})
		if err := e.raw.RPush(context.Background(), key, string(row)).Err(); err != nil {
			t.Fatalf("seed queue: %v", err)
		}
	}
}

type queueItem struct {
	ItemID uuid.UUID
	SongID uuid.UUID
}

func songID(id int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", id))
}

func (e *testEnv) subscribe() *Listener {
	sh := e.hub.GetOrCreate(e.streamID, 16)
	l := newTestListener(sh)
	sh.Subscribe(l)
	return l
}

func (e *testEnv) isActive() bool {
	active, err := e.sredis.IsActive(context.Background(), e.streamID)
	if err != nil {
		return false
	}
	return active
}

// --- control-message waiters ---

func parseCtrl(msg string) map[string]string {
	var m map[string]string
	_ = json.Unmarshal([]byte(msg), &m)
	return m
}

func (e *testEnv) waitForSong(t *testing.T, l *Listener, want uuid.UUID) map[string]string {
	t.Helper()
	return e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "song" && m["songId"] == want.String()
	})
}

func (e *testEnv) collectUntil(t *testing.T, l *Listener, done func(map[string]string) bool) map[string]string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case m, ok := <-l.Send:
			if !ok || m.Ctrl == "" {
				continue // ignore audio chunks on the unified channel
			}
			parsed := parseCtrl(m.Ctrl)
			if done(parsed) {
				return parsed
			}
		default:
			if !e.isActive() && len(l.Send) == 0 {
				// avoid spinning forever after the stream self-deactivated
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	buf := drainControl(l)
	t.Fatalf("timed out waiting for control message (%d drained: %+v)", len(buf), buf)
	return nil
}

func (e *testEnv) waitChunk(t *testing.T, l *Listener) []byte {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case m, ok := <-l.Send:
			if !ok || m.Chunk == nil {
				continue // ignore control frames on the unified channel
			}
			return m.Chunk
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("timed out waiting for audio chunk")
	return nil
}

func TestNextEntry(t *testing.T) {
	entries := []senderredis.QueueEntry{
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SongID: songID(1)},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), SongID: songID(2)},
		{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), SongID: songID(3)},
	}

	cases := []struct {
		name     string
		cursor   uuid.UUID
		wantSong int
		wantNil  bool
	}{
		{"first -> second", entries[0].ItemID, 2, false},
		{"second -> third", entries[1].ItemID, 3, false},
		{"last -> nil", entries[2].ItemID, 0, true},
		{"unknown cursor -> head", uuid.New(), 1, false},
		{"empty queue", uuid.New(), 0, true},
	}

	t.Run("with entries", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.name == "empty queue" {
					n := nextEntry(nil, tc.cursor)
					if tc.wantNil && n != nil {
						t.Fatal("expected nil")
					}
					return
				}
				n := nextEntry(entries, tc.cursor)
				if tc.wantNil {
					if n != nil {
						t.Fatalf("expected nil, got %+v", n)
					}
					return
				}
				if n == nil || n.SongID != songID(tc.wantSong) {
					t.Fatalf("next = %+v, want song %d", n, tc.wantSong)
				}
			})
		}
	})

	t.Run("empty queue", func(t *testing.T) {
		if n := nextEntry(nil, uuid.New()); n != nil {
			t.Fatalf("expected nil for empty queue, got %+v", n)
		}
	})
}

func TestServiceStart_QueueEmpty(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(1000)})
	if err := e.svc.Start(e.streamID); err == nil {
		t.Fatal("Start on empty queue should fail")
	}
}

// TestServeStreamPacing_ReleasesSecondChunkByItsStart guards the per-song
// pacing regression: chunk 2 must be released about one chunk-duration after
// chunk 1 (chunkDur - pacingLead), not a full chunk later (which idles the
// client ~3.3 s into every song). A 16384 B chunk is ~1.02 s of 128 kbps audio
// (> the 800 ms pacing lead). Buggy math shipped chunk 2 at ~1.25 s; the fixed
// pacing ships it at ~0.25 s.
func TestServeStreamPacing_ReleasesSecondChunkByItsStart(t *testing.T) {
	const chunkSize = int64(16384)
	e := newTestEnvChunkSize(t, testAudio(2000), chunkSize)
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)

	// Record wall-clock arrivals of the first two audio chunks.
	var times []time.Time
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && len(times) < 2 {
		select {
		case m, ok := <-l.Send:
			if !ok || m.Chunk == nil {
				continue
			}
			times = append(times, time.Now())
		default:
			time.Sleep(2 * time.Millisecond)
		}
	}
	if len(times) < 2 {
		t.Fatalf("received only %d chunk(s)", len(times))
	}
	gap := times[1].Sub(times[0])
	// Chunk is 40 frames (16680 B, ~1.045 s of audio). Fixed pacing gap ≈
	// 1.045 s − 800 ms ≈ 245 ms. Buggy pacing gap ≈ 2·1.045 s − 800 ms ≈ 1.29 s.
	if gap < 40*time.Millisecond || gap > 600*time.Millisecond {
		t.Fatalf("gap between chunk 1 and 2 = %v, want ≈ 245 ms (chunk2 must be released by the start of its own audio, not its end)", gap)
	}
	_ = e.svc.Stop(e.streamID)
}

func TestServiceStart_DeliversSongAndChunks(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(2000)})
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)
	e.waitChunk(t, l)

	if !e.isActive() {
		t.Fatal("stream should be active after Start")
	}
	sh := e.hub.Get(e.streamID)
	if sh == nil || sh.SongID != s1 {
		t.Fatalf("hub song = %v, want %s", sh, s1)
	}
	// data actually delivered to the listener
	if sh.BytesSent <= 0 {
		t.Fatal("no bytes were streamed")
	}
	_ = e.svc.Stop(e.streamID)
}

func TestServiceSkip_AdvancesSong(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(3000), songID(2).String(): testAudio(3000)})
	s1, s2 := songID(1), songID(2)
	e.seedQueue(t,
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SongID: s1},
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), SongID: s2},
	)
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)
	e.waitChunk(t, l)

	if _, err := e.svc.Skip(e.streamID); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	// song_ended(s1) must precede song(s2) on the control channel
	e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "song" && m["songId"] == s2.String()
	})
	e.waitChunk(t, l)

	if sh := e.hub.Get(e.streamID); sh.SongID != s2 {
		t.Fatalf("hub song after skip = %s, want %s", sh.SongID, s2)
	}
	_ = e.svc.Stop(e.streamID)
}

func TestServiceSkip_AutoStopAtQueueEnd(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(4000)})
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)
	e.waitChunk(t, l)

	if next, err := e.svc.Skip(e.streamID); err != nil {
		t.Fatalf("Skip: %v", err)
	} else if next != nil {
		t.Fatalf("Skip returned %v, want nil on exhausted queue", next)
	}

	e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "stream_ended"
	})
	if e.isActive() {
		t.Fatal("stream should be deactivated after exhausting the queue")
	}
}

func TestServiceNaturalAdvance_EndsAndDeactivates(t *testing.T) {
	// Tiny audio (few frames, under one chunk) makes each song EOF immediately,
	// so the full start -> advance -> exhausted lifecycle runs deterministically.
	tiny := testAudio(3)
	e := newTestEnv(t, map[string][]byte{songID(1).String(): tiny, songID(2).String(): tiny})
	s1, s2 := songID(1), songID(2)
	e.seedQueue(t,
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SongID: s1},
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), SongID: s2},
	)
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)
	// song1 ends -> advance to song2
	e.waitForSong(t, l, s2)
	// song2 ends -> queue exhausted -> stream_ended
	e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "stream_ended"
	})

	if e.isActive() {
		t.Fatal("stream should be deactivated after natural end of the last song")
	}
	cursor, _ := e.sredis.GetCursor(context.Background(), e.streamID)
	if cursor != uuid.Nil {
		t.Fatalf("cursor should be cleared, got %s", cursor)
	}
}

func TestServiceNaturalAdvance_EmitsSongEndedBeforeNextSong(t *testing.T) {
	tiny := testAudio(3)
	e := newTestEnv(t, map[string][]byte{songID(1).String(): tiny, songID(2).String(): tiny})
	s1, s2 := songID(1), songID(2)
	e.seedQueue(t,
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SongID: s1},
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), SongID: s2},
	)
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)

	wantEnded := false
	e.collectUntil(t, l, func(m map[string]string) bool {
		if m["type"] == "song_ended" && m["songId"] == s1.String() {
			wantEnded = true
		}
		return m["type"] == "song" && m["songId"] == s2.String()
	})
	if !wantEnded {
		t.Fatal("song_ended(s1) was not delivered before song(s2)")
	}
	_ = e.svc.Stop(e.streamID)
}

func TestServiceStop_DeactivatesAndNotifies(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(2000)})
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitForSong(t, l, s1)

	if err := e.svc.Stop(e.streamID); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "stream_stopped"
	})
	if e.isActive() {
		t.Fatal("stream still active after Stop")
	}
	if sh := e.hub.Get(e.streamID); sh != nil && sh.SongID != uuid.Nil {
		t.Fatalf("hub still has song %s after Stop", sh.SongID)
	}
}

func TestServiceStatus(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(1000)})
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})

	st, err := e.svc.Status(context.Background(), e.streamID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.QueueLength != 1 || st.IsActive {
		t.Fatalf("status before start = %+v", st)
	}

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	st, err = e.svc.Status(context.Background(), e.streamID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.IsActive || st.CurrentSongID == nil || *st.CurrentSongID != s1.String() {
		t.Fatalf("status after start = %+v", st)
	}
	if st.Position == nil || *st.Position != 0 {
		t.Fatalf("position = %+v, want 0", st.Position)
	}
	_ = e.svc.Stop(e.streamID)
}

func TestWorker_HandleEvent(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(1000)})
	s1 := songID(1)
	e.seedQueue(t, queueItem{ItemID: e.streamID, SongID: s1})

	ctx := context.Background()
	if err := e.sredis.EnsureConsumerGroup(ctx, e.streamID); err != nil {
		t.Fatalf("ensure group: %v", err)
	}
	if err := e.sredis.SetActive(ctx, e.streamID); err != nil {
		t.Fatalf("set active: %v", err)
	}
	l := e.subscribe()
	w := NewWorker(e.svc, e.sredis)

	payload, _ := json.Marshal(songPayload{ItemID: e.streamID.String(), SongID: s1.String()})
	w.handleEvent(ctx, e.streamID, &senderredis.EventMessage{ID: "1-0", Type: "stream_started", Payload: payload})
	if !e.isActive() {
		t.Fatal("worker stream_started did not activate the stream")
	}
	e.waitForSong(t, l, s1)

	// Production order: stream-service clears runtime keys, then publishes
	// stream_stopped; the worker only stops playback and notifies listeners.
	if err := e.sredis.Deactivate(ctx, e.streamID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	w.handleEvent(ctx, e.streamID, &senderredis.EventMessage{ID: "2-0", Type: "stream_stopped"})
	e.collectUntil(t, l, func(m map[string]string) bool {
		return m["type"] == "stream_stopped"
	})
	if e.isActive() {
		t.Fatal("stream should remain inactive after worker stream_stopped")
	}
	if sh := e.hub.Get(e.streamID); sh == nil || sh.SongID != uuid.Nil {
		t.Fatal("hub playback not cleared after worker stream_stopped")
	}
}

// TestService_CancelledServeStreamStopsStreaming guards the transition path:
// after a Skip cancels the old serve goroutine, no stale chunks may be emitted.
func TestService_SkipDoesNotEmitDeletedSongChunks(t *testing.T) {
	e := newTestEnv(t, map[string][]byte{songID(1).String(): testAudio(100), songID(2).String(): testAudio(100)})
	s1, s2 := songID(1), songID(2)
	e.seedQueue(t,
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), SongID: s1},
		queueItem{ItemID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), SongID: s2},
	)
	l := e.subscribe()

	if err := e.svc.Start(e.streamID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	e.waitChunk(t, l)

	// Small audio EOFs quickly; run the skip while song1 may still be mid-flight.
	if _, err := e.svc.Skip(e.streamID); err != nil {
		t.Fatalf("Skip: %v", err)
	}
	e.waitForSong(t, l, s2)

	sh := e.hub.Get(e.streamID)
	if sh.SongID != s2 {
		t.Fatalf("hub song = %s, want %s", sh.SongID, s2)
	}
	_ = e.svc.Stop(e.streamID)
}

func TestFetchChunk_ContentRangeParsing(t *testing.T) {
	audio := testAudio(5)
	server := httptest.NewServer(&fakeContent{files: map[string][]byte{"s1": audio}, chunkSize: testChunkSize})
	defer server.Close()

	c := contenthttp.NewContentClient(server.URL, testChunkSize)
	res, err := c.FetchChunk("s1", 0, "owner")
	if err != nil {
		t.Fatalf("FetchChunk: %v", err)
	}
	want := int64(len(audio))
	if res.FileSize != want || len(res.Data) == 0 {
		t.Fatalf("res = len(data)=%d filesize=%d (want %d)", len(res.Data), res.FileSize, want)
	}

	// offset beyond EOF -> 416 path must yield a FileSize with nil data
	res, err = c.FetchChunk("s1", 1<<20, "owner")
	if err != nil {
		t.Fatalf("FetchChunk past EOF: %v", err)
	}
	if res.Data != nil || res.FileSize != want {
		t.Fatalf("past-EOF res = %+v", res)
	}
}
