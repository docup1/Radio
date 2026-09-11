package application

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestListener(sh *StreamHub) *Listener {
	if sh.Listeners == nil {
		sh.Listeners = make(map[*Listener]struct{})
	}
	if sh.MaxCap == 0 {
		sh.MaxCap = 16
	}
	return &Listener{
		ID:        uuid.NewString(),
		Ch:        make(chan []byte, 64),
		ControlCh: make(chan string, 64),
		Hub:       sh,
	}
}

func drainControl(l *Listener) []map[string]string {
	var out []map[string]string
	for {
		select {
		case msg := <-l.ControlCh:
			var m map[string]string
			_ = json.Unmarshal([]byte(msg), &m)
			out = append(out, m)
		default:
			return out
		}
	}
}

func TestBeginSongResetsBuffer(t *testing.T) {
	sh := &StreamHub{MaxCap: 32}
	sh.AddChunk([]byte("old"))

	sh.BeginSong(uuid.New(), time.Now().UnixNano())
	// BeginSong must clear the ring buffer and reset counters.
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if len(sh.Chunks) != 0 {
		t.Fatalf("Chunks = %d, want 0 after BeginSong", len(sh.Chunks))
	}
	if sh.BytesSent != 0 || sh.FileSize != 0 {
		t.Fatal("counters not reset by BeginSong")
	}
	if !sh.Active {
		t.Fatal("BeginSong should mark the stream active")
	}
}

func TestAddChunkRingBufferCap(t *testing.T) {
	sh := &StreamHub{MaxCap: 8}
	sh.Active = true
	for i := 0; i < 20; i++ {
		sh.AddChunk([]byte{byte(i)})
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if len(sh.Chunks) != 8 {
		t.Fatalf("Chunks retained %d, want MaxCap 8", len(sh.Chunks))
	}
	if sh.Chunks[0][0] != 12 {
		t.Fatalf("ring buffer head = %d, want oldest survivor 12", sh.Chunks[0][0])
	}
}

func TestAddChunkFanOut(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	sh.Active = true
	l1 := newTestListener(sh)
	l2 := newTestListener(sh)
	sh.Subscribe(l1)
	sh.Subscribe(l2)

	sh.AddChunk([]byte("data1"))
	sh.AddChunk([]byte("data2"))

	for _, l := range []*Listener{l1, l2} {
		if got := <-l.Ch; string(got) != "data1" {
			t.Fatalf("listener got %q first, want data1", got)
		}
		if got := <-l.Ch; string(got) != "data2" {
			t.Fatalf("listener got %q last, want data2", got)
		}
	}
}

func TestAddChunkIgnoredWhenInactive(t *testing.T) {
	sh := &StreamHub{MaxCap: 8}
	l := newTestListener(sh)
	sh.Subscribe(l)

	sh.Active = false
	sh.AddChunk([]byte("x"))
	sh.mu.Lock()
	if len(sh.Chunks) != 0 {
		t.Fatal("chunk added while stream inactive")
	}
	sh.mu.Unlock()
	select {
	case <-l.Ch:
		t.Fatal("chunk delivered while stream inactive")
	default:
	}
}

func TestAddChunkSlowListenerDrop(t *testing.T) {
	// A listener whose channel is full must not block the hub.
	sh := &StreamHub{MaxCap: 16, Active: true}
	sh.Listeners = make(map[*Listener]struct{})
	l := &Listener{
		ID:        uuid.NewString(),
		Ch:        make(chan []byte, 1),
		ControlCh: make(chan string, 1),
		Hub:       sh,
	}
	sh.Subscribe(l)
	l.Ch <- []byte("full")
	for i := 0; i < 100; i++ {
		sh.AddChunk([]byte{byte(i)})
	}
}

func TestSubscribeBackfillAndAnnounce(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	sh.Active = true
	songID := uuid.New()
	sh.BeginSong(songID, time.Now().UnixNano())
	sh.AddChunk([]byte("latest"))

	l := newTestListener(sh)
	sh.Subscribe(l)

	if got := <-l.Ch; string(got) != "latest" {
		t.Fatalf("backfill = %q, want latest chunk", got)
	}
	ctrl := drainControl(l)
	if len(ctrl) != 1 || ctrl[0]["type"] != "song" {
		t.Fatalf("expected song announce, got %+v", ctrl)
	}
	if ctrl[0]["songId"] != songID.String() {
		t.Fatalf("announce songId = %s, want %s", ctrl[0]["songId"], songID)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	sh.Active = true
	l := newTestListener(sh)
	sh.Subscribe(l)
	sh.Unsubscribe(l)

	sh.AddChunk([]byte("x"))
	select {
	case <-l.Ch:
		t.Fatal("chunk delivered after Unsubscribe")
	default:
	}
}

func TestSendControlFanOut(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	l1 := newTestListener(sh)
	l2 := newTestListener(sh)
	sh.Subscribe(l1)
	sh.Subscribe(l2)

	sh.SendControl([]byte(`{"type":"x"}`))
	for _, l := range []*Listener{l1, l2} {
		if got := <-l.ControlCh; got != `{"type":"x"}` {
			t.Fatalf("control = %q", got)
		}
	}
}

func TestSendControlToOne(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	l1 := newTestListener(sh)
	l2 := newTestListener(sh)
	sh.Subscribe(l1)
	sh.Subscribe(l2)

	sh.SendControlToOne(l1, []byte(`{"type":"y"}`))
	if got := <-l1.ControlCh; got != `{"type":"y"}` {
		t.Fatalf("l1 control = %q", got)
	}
	select {
	case <-l2.ControlCh:
		t.Fatal("l2 received one-to-one message")
	default:
	}
}

func TestStopPlaybackCancelsAndClears(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	ctx, cancel := context.WithCancel(context.Background())
	sh.SetCancel(cancel)
	sh.BeginSong(uuid.New(), time.Now().UnixNano())
	sh.StopPlayback()

	select {
	case <-ctx.Done():
	default:
		t.Fatal("StopPlayback did not cancel the serve goroutine")
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.Active {
		t.Fatal("StopPlayback left stream active")
	}
	if sh.SongID != uuid.Nil {
		t.Fatal("StopPlayback did not clear SongID")
	}
	if len(sh.Chunks) != 0 {
		t.Fatal("StopPlayback did not clear chunks")
	}
}

func TestCancelInvokesContextCancel(t *testing.T) {
	sh := &StreamHub{MaxCap: 16}
	ctx, cancel := context.WithCancel(context.Background())
	sh.SetCancel(cancel)
	sh.Cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("Cancel did not cancel the serve context")
	}
}

func TestHubRemoveClosesListeners(t *testing.T) {
	h := NewHub()
	sh := h.GetOrCreate(uuid.New(), 16)
	l := newTestListener(sh)
	sh.Subscribe(l)

	h.Remove(sh.StreamID)
	select {
	case _, ok := <-l.Ch:
		if ok {
			t.Fatal("listener channel not closed on hub Remove")
		}
	default:
		t.Fatal("listener channel not closed on hub Remove")
	}
	if h.Get(sh.StreamID) != nil {
		t.Fatal("stream hub still present after Remove")
	}
}

func TestHubGetOrCreateReuses(t *testing.T) {
	h := NewHub()
	id := uuid.New()
	a := h.GetOrCreate(id, 16)
	b := h.GetOrCreate(id, 16)
	if a != b {
		t.Fatal("GetOrCreate returned a different hub for the same id")
	}
}

func BenchmarkAddChunk(b *testing.B) {
	sh := &StreamHub{MaxCap: 64, Active: true}
	for i := 0; i < 10; i++ {
		sh.Subscribe(newTestListener(sh))
	}
	chunk := bytes.Repeat([]byte{1}, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sh.AddChunk(chunk)
	}
}
