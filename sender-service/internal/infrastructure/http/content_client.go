package http

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// ContentClient fetches audio chunks from the content-service.
type ContentClient struct {
	baseURL   string
	client    *http.Client
	chunkSize int64
}

// NewContentClient creates a client that talks to content-service.
func NewContentClient(baseURL string, chunkSize int64) *ContentClient {
	return &ContentClient{
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		chunkSize: chunkSize,
	}
}

// FetchChunkResult contains the audio data and metadata for a chunk.
type FetchChunkResult struct {
	Data     []byte
	FileSize int64
}

// FetchChunk retrieves a chunk of audio for songID starting at offset.
// ownerID is passed as X-Owner-ID header (required by content-service for access control).
// Returns the raw bytes and total file size.
func (c *ContentClient) FetchChunk(songID string, offset int64, ownerID string) (*FetchChunkResult, error) {
	url := fmt.Sprintf("%s/songs/%s/audio", c.baseURL, songID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Owner-ID", ownerID)
	end := offset + c.chunkSize - 1
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 416 Range Not Satisfiable = offset >= file size (EOF)
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		var fileSize int64
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if idx := len(cr) - 1; idx > 0 {
				for idx > 0 && cr[idx] != '/' {
					idx--
				}
				if idx > 0 {
					fileSize, _ = strconv.ParseInt(cr[idx+1:], 10, 64)
				}
			}
		}
		return &FetchChunkResult{Data: nil, FileSize: fileSize}, nil
	}
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("content-service: unexpected status %d", resp.StatusCode)
	}

	// Limit reads to chunkSize+1 to avoid reading entire file if server
	// returns 200 instead of 206 by mistake.
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.chunkSize+1))
	if err != nil {
		return nil, err
	}

	// Parse Content-Range to get total file size: "bytes 0-65535/12345678"
	var fileSize int64
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		// format: "bytes <start>-<end>/<total>"
		if idx := len(cr) - 1; idx > 0 {
			for idx > 0 && cr[idx] != '/' {
				idx--
			}
			if idx > 0 {
				fileSize, _ = strconv.ParseInt(cr[idx+1:], 10, 64)
			}
		}
	} else {
		fileSize = resp.ContentLength
	}

	return &FetchChunkResult{Data: data, FileSize: fileSize}, nil
}
