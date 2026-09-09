package main

import (
	"fmt"
	"os"

	"radio/sender-service/internal/media"
)

func main() {
	for _, f := range []string{"/Users/docup/Projects/Radio/test-fixtures/song.mp3", "/Users/docup/Projects/Radio/test-fixtures/song2.mp3"} {
		b, err := os.ReadFile(f)
		if err != nil {
			fmt.Println(err)
			continue
		}
		p := media.NewMP3Parser()
		frames := p.Feed(b)
		tail := p.Flush()
		fmt.Printf("%s: bytes=%d frames=%d tail=%d dur=%v (%d ns/frame)\n", f, len(b), len(frames), len(tail), p.Dur, int64(p.Dur)/int64(max(1, len(frames))))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
