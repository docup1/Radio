package media

import "time"

// MP3Parser splits a byte stream into MPEG-audio-frame-aligned buffers.
//
// Fetching audio in fixed 64 KiB slices cuts the MP3 inside MPEG frames, which
// makes browsers glitch (they re-sync on the next frame and drop the partial
// audio). This parser holds back incomplete frames and only emits whole
// MPEG audio frames, so every buffer sent over the WebSocket starts and ends
// on a frame boundary.
type MP3Parser struct {
	buf []byte

	// Dur is the total playback duration of every frame emitted so far.
	// It lets the streaming loop pace chunk delivery to the song's real
	// timeline instead of dumping the whole file at once.
	Dur time.Duration
}

// NewMP3Parser creates an empty parser.
func NewMP3Parser() *MP3Parser {
	return &MP3Parser{}
}

// Feed appends data and returns every complete MPEG frame found so far.
// It is safe to call with arbitrary chunk boundaries; frames that straddle
// two feeds are held until complete.
func (p *MP3Parser) Feed(data []byte) [][]byte {
	p.buf = append(p.buf, data...)

	var frames [][]byte
	for {
		if len(p.buf) < 4 {
			break
		}

		// Resist false sync by scanning for a valid header start.
		idx := findSync(p.buf)
		if idx < 0 {
			// No sync in sight. Keep at most 3 bytes (a partial header) so we
			// don't grow the buffer forever with pure garbage.
			if len(p.buf) > 3 {
				p.buf = p.buf[len(p.buf)-3:]
			}
			break
		}
		if idx > 0 {
			p.buf = p.buf[idx:]
		}
		if len(p.buf) < 4 {
			break
		}

		flen := frameLength(p.buf)
		if flen < 4 || flen > 4096 {
			// Not a real frame — skip one byte and resync.
			p.buf = p.buf[1:]
			continue
		}
		if len(p.buf) < flen {
			break // wait for the rest of this frame
		}

		frames = append(frames, append([]byte(nil), p.buf[:flen]...))
		p.Dur += FrameDuration(p.buf)
		p.buf = p.buf[flen:]
	}
	return frames
}

// Flush returns any trailing bytes that could not form a complete frame
// (e.g. the final partial frame at EOF). The caller may still send these.
func (p *MP3Parser) Flush() []byte {
	tail := p.buf
	p.buf = nil
	return tail
}

// FrameDuration returns the playback duration of the MPEG audio frame
// starting at b[0:4] (samples per frame / sample rate). It is the exported
// counterpart used by the streaming loop to track chunk pacing positions.
func FrameDuration(b []byte) time.Duration {
	ver := (b[1] >> 3) & 0x03 // 0=MPEG2.5, 1=reserved, 2=MPEG2, 3=MPEG1
	layer := (b[1] >> 1) & 0x03
	if ver == 1 {
		return 0
	}
	samples := 1152
	switch layer {
	case 3: // Layer I
		samples = 384
	case 2: // Layer II
	default: // Layer III
	}
	if ver == 0 || ver == 2 { // MPEG2 / MPEG2.5 → half-size frames
		samples /= 2
	}
	sri := int((b[2] >> 2) & 0x03)
	sr := sampleRates[ver][sri]
	if sr == 0 {
		return 0
	}
	return time.Duration(samples) * time.Second / time.Duration(sr)
}

// findSync returns the index of the first MPEG sync word (0xFF Ex).
func findSync(b []byte) int {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == 0xFF && (b[i+1]&0xE0) == 0xE0 {
			return i
		}
	}
	return -1
}

// frameLength computes the byte length of the MPEG frame starting at b[0:4]
// (the 32-bit frame header). Returns 0 for headers that are not decodable.
func frameLength(b []byte) int {
	ver := (b[1] >> 3) & 0x03 // 0=MPEG2.5, 1=reserved, 2=MPEG2, 3=MPEG1
	layer := (b[1] >> 1) & 0x03
	if ver == 1 || layer == 0 {
		return 0
	}

	bri := int(b[2] >> 4)
	sri := int((b[2] >> 2) & 0x03)
	padding := int((b[2] >> 1) & 0x01)
	if bri == 0 || bri == 15 { // free or invalid bitrate
		return 0
	}

	sr := sampleRates[ver][sri]
	br := bitrates[ver][bri]
	if sr == 0 || br == 0 {
		return 0
	}

	if layer == 3 {
		// Layer I: 12 * bitrate / sampleRate + padding, times 4.
		return (br*12000/sr + padding) * 4
	}

	samples := 1152
	if ver == 0 || ver == 2 { // MPEG2 / MPEG2.5 → half-size frames
		samples = 576
	}
	return samples*br*1000/(8*sr) + padding
}

// bitrate tables in kbps indexed by version (0=2.5, 2=MPEG2, 3=MPEG1).
// Only Layer III values are required here; anything else is 0.
var bitrates = map[byte][16]int{
	3: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
	2: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
	0: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
}

// sampleRates indexed by version then by the 2-bit sample-rate index.
// Index 3 is "reserved" (0).
var sampleRates = map[byte][4]int{
	3: {44100, 48000, 32000, 0},
	2: {22050, 24000, 16000, 0},
	0: {11025, 12000, 8000, 0},
}
