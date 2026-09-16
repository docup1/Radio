package media

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// frameLenMPEG1L3 is the byte length of an MPEG1 Layer III frame at
// 128 kbps / 44.1 kHz without padding: 1152*128000/(8*44100) + 0 = 417.
const frameLenMPEG1L3 = 417

// synthFrame builds a valid MPEG1 Layer III frame header followed by a
// zero-filled payload sized to the length the parser computes from that
// header (so length and bitrate/samplerate always agree).
func synthFrame(bitrateIdx, sampleRateIdx, padding int) []byte {
	header := make([]byte, 4)
	header[0] = 0xFF
	header[1] = 0xFB // MPEG1, Layer III, no CRC
	br := byte(bitrateIdx)
	sr := byte(sampleRateIdx)
	header[2] = br<<4 | sr<<2
	if padding == 1 {
		header[2] |= 0x02
	}
	header[3] = 0x00

	n := frameLength(header)
	if n < 4 {
		panic("synthFrame: unreachable frame length")
	}
	f := make([]byte, n)
	copy(f, header)
	for i := 4; i < len(f); i++ {
		f[i] = 0xCC
	}
	return f
}

func frameDur(t *testing.T, b []byte) time.Duration {
	t.Helper()
	d := FrameDuration(b)
	if d <= 0 {
		t.Fatalf("frameDuration returned non-positive %v", d)
	}
	return d
}

func TestFeed_WholeFrame(t *testing.T) {
	p := NewMP3Parser()
	frame := synthFrame(9, 0, 0)

	frames := p.Feed(frame)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if !bytes.Equal(frames[0], frame) {
		t.Fatal("emitted frame does not match input")
	}
	want := time.Duration(1152) * time.Second / 44100
	if p.Dur != want {
		t.Fatalf("Dur = %v, want %v", p.Dur, want)
	}
	if tail := p.Flush(); len(tail) != 0 {
		t.Fatalf("Flush returned %d bytes after a clean frame", len(tail))
	}
}

func TestFeed_SplitAcrossBoundaries(t *testing.T) {
	p := NewMP3Parser()
	frame := synthFrame(9, 0, 0)

	// Feed the frame one byte at a time; it must materialize only once.
	var emitted [][]byte
	for i := 0; i < len(frame); i++ {
		emitted = append(emitted, p.Feed(frame[i:i+1])...)
	}
	if len(emitted) != 1 {
		t.Fatalf("expected exactly 1 frame after byte-by-byte feed, got %d", len(emitted))
	}
	if !bytes.Equal(emitted[0], frame) {
		t.Fatal("byte-by-byte frame mismatch")
	}
}

func TestFeed_MultipleFramesAndResync(t *testing.T) {
	p := NewMP3Parser()

	// Garbage prefix followed by 2 clean frames followed by garbage tail.
	junk := []byte{0x00, 0x11, 0x22, 0x33, 0xFF, 0x00}
	f1 := synthFrame(9, 0, 0)
	f2 := synthFrame(10, 0, 0)

	in := append(append(append(append([]byte{}, junk...), f1...), f2...), 0xAB, 0xCD)
	frames := p.Feed(in)
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames after resync, got %d", len(frames))
	}
	if !bytes.Equal(frames[0], f1) {
		t.Fatal("first frame mismatch after resync")
	}
	if !bytes.Equal(frames[1], f2) {
		t.Fatal("second frame mismatch after resync")
	}
	// Trailing garbage stays in the buffer; Flush returns it.
	tail := p.Flush()
	if len(tail) != 2 || tail[0] != 0xAB || tail[1] != 0xCD {
		t.Fatalf("Flush returned %v, want [0xAB 0xCD]", tail)
	}
}

func TestFeed_PaddedFrameDurationAndLength(t *testing.T) {
	// Frame with padding bit set must be 1 byte longer.
	plain := synthFrame(9, 0, 0)
	padded := synthFrame(9, 0, 1)
	if len(padded) != len(plain)+1 {
		t.Fatalf("padded frame len = %d, want %d", len(padded), len(plain)+1)
	}

	p := NewMP3Parser()
	got := p.Feed(padded)
	if len(got) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(got))
	}
	wantDur := time.Duration(1152) * time.Second / 44100
	if p.Dur != wantDur {
		t.Fatalf("Dur with padding = %v, want %v", p.Dur, wantDur)
	}
}

func TestFeed_PureJunkDoesNotGrow(t *testing.T) {
	p := NewMP3Parser()
	junk := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	frames := p.Feed(junk)
	if len(frames) != 0 {
		t.Fatalf("expected no frames from junk, got %d", len(frames))
	}
	// Buffer must stay bounded: at most 3 bytes (partial header) are kept.
	if len(p.buf) > 3 {
		t.Fatalf("parser buffer grew to %d bytes on pure junk", len(p.buf))
	}
}

func TestFeed_DurIsCumulative(t *testing.T) {
	p := NewMP3Parser()
	f1 := synthFrame(9, 0, 0)
	f2 := synthFrame(11, 1, 0) // 160kbps, 48kHz
	got := p.Feed(append(f1, f2...))
	if len(got) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(got))
	}
	want := frameDur(t, f1) + frameDur(t, f2)
	if p.Dur != want {
		t.Fatalf("Dur = %v, want %v", p.Dur, want)
	}
}

func TestFeed_InvalidHeaderResyncs(t *testing.T) {
	// A false sync in the middle (0xFF Ex with garbage) is skipped and parsing
	// resumes at the first real frame.
	p := NewMP3Parser()
	f := synthFrame(9, 0, 0)
	in := append([]byte{0xFF, 0xF0, 0xAA, 0xAA, 0xBB, 0xBB}, f...)
	frames := p.Feed(in)
	if len(frames) != 1 {
		t.Fatalf("expected 1 real frame, got %d", len(frames))
	}
	if !bytes.Equal(frames[0], f) {
		t.Fatal("resynced frame mismatch")
	}
}

// TestFeed_RealFixture exercises the parser against the committed MP3 files
// (moved into tests/fixtures/audio). It is skipped when fixtures are absent
// so small checkouts and plain `go test ./internal/...` still pass.
func TestFeed_RealFixture(t *testing.T) {
	path := fixturePath(t, "song.mp3")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	p := NewMP3Parser()
	sent := 0
	for off := 0; off < len(data); off += 65536 {
		end := off + 65536
		if end > len(data) {
			end = len(data)
		}
		frames := p.Feed(data[off:end])
		for _, f := range frames {
			if len(f) < 4 {
				t.Fatalf("frame shorter than header")
			}
			if f[0] != 0xFF || f[1]&0xE0 != 0xE0 {
				t.Fatalf("frame does not start with MPEG sync")
			}
			sent += len(f)
		}
	}
	if p.Dur <= 0 {
		t.Fatal("parser detected no audio duration in real fixture")
	}
	if sent <= 0 {
		t.Fatal("no whole frames were produced from the real fixture")
	}
	// No infinite loop, buffer drained to a tail <= 3 garbage bytes.
	if len(p.buf) > 3 {
		t.Fatalf("parser buffer retained %d bytes after full file", len(p.buf))
	}
}

// TestFeed_BytesPerSecond sanity-check: the duration implied by the parsed
// frames must be consistent with the size of a 128kbps stream.
func TestFeed_BytesPerSecond(t *testing.T) {
	p := NewMP3Parser()
	for total := 0; total < 4*1024; total += frameLenMPEG1L3 {
		_ = p.Feed(synthFrame(9, 0, 0))
	}
	bytes := p.Dur.Seconds() * (128000 / 8)
	if bytes < 4000 || bytes > 4200 {
		t.Fatalf("byte rate mismatch: %d bytes expected in %v of audio", int(bytes), p.Dur)
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := cwd
	for i := 0; i < 12; i++ {
		p := filepath.Join(dir, "tests", "fixtures", "audio", name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skipf("fixture %s not found (searched up from %s)", name, cwd)
	return ""
}
