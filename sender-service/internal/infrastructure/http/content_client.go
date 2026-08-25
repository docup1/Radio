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

// FetchChunk retrieves the full audio for songID.
// ownerID is passed as X-Owner-ID header (required by content-service for access control).
// Returns the full audio bytes for decoding by the client.
func (c *ContentClient) FetchChunk(songID string, offset int64, ownerID string) ([]byte, error) {
	url := fmt.Sprintf("%s/songs/%s/audio", c.baseURL, songID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Owner-ID", ownerID)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("content-service: unexpected status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ClearETag removes the cached ETag for a song (e.g. on song change).
func (c *ContentClient) ClearETag(songID string) {
	delete(c.etagCache, songID)
}
