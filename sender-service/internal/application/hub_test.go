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
		ID:   uuid.NewString(),
		Send: make(chan OutMsg, 64),
		Hub:  sh,
	}
}

// recvChunk reads the next audio chunk from the unified Send channel.
func recvChunk(t *testing.T, l *Listener) []byte {
	t.Helper()
	m, ok := <-l.Send
	if !ok {
		t.Fatal("listener channel closed unexpectedly")
	}
	if m.Chunk == nil {
		t.Fatalf("expected audio chunk, got control frame: %s", m.Ctrl)
	}
	return m.Chunk
}

// recvControl reads the next control message from the unified Send channel.
func recvControl(t *testing.T, l *Listener) map[string]string {
	t.Helper()
	m, ok := <-l.Send
	if !ok {
		t.Fatal("listener channel closed unexpectedly")
	}
	if m.Ctrl == "" {
		t.Fatal("expected control frame, got audio chunk")
	}
	var out map[string]string
	_ = json.Unmarshal([]byte(m.Ctrl), &out)
	return out
}

// drainControl returns all pending control messages on the Send channel.
func drainControl(l *Listener) []map[string]string {
	var out []map[string]string
	for {
		select {
		case m := <-l.Send:
			if m.Ctrl == "" {
				continue // skip audio chunks
			}
			var msg map[string]string
			_ = json.Unmarshal([]byte(m.Ctrl), &msg)
			out = append(out, msg)
		default:
			return out
		}
	}
}

// drainSend returns all pending messages (chunk or control) from the Send
// channel, separating them into audio chunks and control frames.
func drainSend(l *Listener) (chunks [][]byte, ctrls []map[string]string) {
	for {
		select {
		case m := <-l.Send:
			if m.Chunk != nil {
				chunks = append(chunks, m.Chunk)
			} else if m.Ctrl != "" {
				var msg map[string]string
				_ = json.Unmarshal([]byte(m.Ctrl), &msg)
				ctrls = append(ctrls, msg)
			}
		default:
			return
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
		if got := recvChunk(t, l); string(got) != "data1" {
			t.Fatalf("listener got %q first, want data1", got)
		}
		if got := recvChunk(t, l); string(got) != "data2" {
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
	case m, ok := <-l.Send:
		if ok && m.Chunk != nil {
			t.Fatal("chunk delivered while stream inactive")
		}
	default:
	}
}

func TestAddChunkSlowListenerDrop(t *testing.T) {
	// A listener whose channel is full must not block the hub.
	sh := &StreamHub{MaxCap: 16, Active: true}
	sh.Listeners = make(map[*Listener]struct{})
	l := &Listener{
		ID:   uuid.NewString(),
		Send: make(chan OutMsg, 1),
		Hub:  sh,
	}
	sh.Subscribe(l)
	l.Send <- OutMsg{Chunk: []byte("full")}
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

	// The unified ordered channel guarantees the backfill chunk arrives BEFORE
	// the song announce — the client must never receive a chunk for a song it
	// has not been told about yet.
	if got := recvChunk(t, l); string(got) != "latest" {
		t.Fatalf("backfill = %q, want latest chunk", got)
	}
	ctrl := recvControl(t, l)
	if ctrl["type"] != "song" {
		t.Fatalf("expected song announce, got %+v", ctrl)
	}
	if ctrl["songId"] != songID.String() {
		t.Fatalf("announce songId = %s, want %s", ctrl["songId"], songID)
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
	case m := <-l.Send:
		if m.Chunk != nil {
			t.Fatal("chunk delivered after Unsubscribe")
		}
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
		if got := recvControl(t, l); got["type"] != "x" {
			t.Fatalf("control = %+v", got)
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
	if got := recvControl(t, l1); got["type"] != "y" {
		t.Fatalf("l1 control = %+v", got)
	}
	select {
	case m := <-l2.Send:
		if m.Ctrl != "" {
			t.Fatal("l2 received one-to-one message")
		}
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

func TestBoundaryControlPrecedesChunks(t *testing.T) {
	// At a song boundary the hub receives song_ended + song control frames and
	// then the new song's audio chunks. The unified channel must deliver them
	// in exactly that order: a chunk for the new song must never be observed
	// before its announce.
	sh := &StreamHub{MaxCap: 16}
	sh.Active = true
	l := newTestListener(sh)
	sh.Subscribe(l)

	sh.SendControl([]byte(`{"type":"song_ended","songId":"a"}`))
	sh.SendControl([]byte(`{"type":"song","songId":"b"}`))
	sh.AddChunk([]byte("b-chunk-1"))
	sh.AddChunk([]byte("b-chunk-2"))

	types := make([]string, 0, 4)
	chunks := make([]string, 0, 2)
	for i := 0; i < 4; i++ {
		m := <-l.Send
		if m.Chunk != nil {
			types = append(types, "chunk")
			chunks = append(chunks, string(m.Chunk))
		} else {
			var msg map[string]string
			_ = json.Unmarshal([]byte(m.Ctrl), &msg)
			types = append(types, msg["type"])
		}
	}
	want := []string{"song_ended", "song", "chunk", "chunk"}
	for i, w := range want {
		if types[i] != w {
			t.Fatalf("order = %v, want %v", types, want)
		}
	}
	if chunks[0] != "b-chunk-1" || chunks[1] != "b-chunk-2" {
		t.Fatalf("chunks = %v", chunks)
	}
}

func TestHubRemoveClosesListeners(t *testing.T) {
	h := NewHub()
	sh := h.GetOrCreate(uuid.New(), 16)
	l := newTestListener(sh)
	sh.Subscribe(l)

	h.Remove(sh.StreamID)
	select {
	case _, ok := <-l.Send:
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
