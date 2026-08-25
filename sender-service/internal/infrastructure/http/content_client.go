package http

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// ContentClient fetches audio chunks from the content-service.
type ContentClient struct {
	baseURL    string
	client     *http.Client
	etagCache  map[string]string // songID → etag
	chunkSize  int64
}

// NewContentClient creates a client that talks to content-service.
func NewContentClient(baseURL string, chunkSize int64) *ContentClient {
	return &ContentClient{
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		etagCache: make(map[string]string),
		chunkSize: chunkSize,
	}
}

// FetchChunk retrieves a chunk of audio for songID starting at offset.
// Returns the raw bytes, or nil if 304 Not Modified (no new data).
func (c *ContentClient) FetchChunk(songID string, offset int64) ([]byte, error) {
	url := fmt.Sprintf("%s/songs/%s/audio", c.baseURL, songID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	end := offset + c.chunkSize - 1
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))

	if etag, ok := c.etagCache[songID]; ok {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if etag := resp.Header.Get("ETag"); etag != "" {
		c.etagCache[songID] = etag
	}

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("content-service: unexpected status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ClearETag removes the cached ETag for a song (e.g. on song change).
func (c *ContentClient) ClearETag(songID string) {
	delete(c.etagCache, songID)
}
