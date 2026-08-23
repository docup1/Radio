package models

import (
	"time"

	"github.com/google/uuid"
)

// Media types accepted by the chunked upload flow.
const (
	MediaTypeAudio = "audio"
)

// Upload session lifecycle states.
const (
	UploadStatusInitialized = "initialized"
	UploadStatusCompleted  = "completed"
	UploadStatusAborted    = "aborted"
)

// UploadSession tracks a resumable chunked upload of a single media file. Chunks
// are persisted on the filesystem; the finalized file is assembled on confirm.
type UploadSession struct {
	ID            uuid.UUID `json:"id"`
	OwnerID       uuid.UUID `json:"owner_id"`
	MediaType     string    `json:"media_type"`
	Status        string    `json:"status"`
	ContentType   string    `json:"content_type"`
	TotalChunks   int       `json:"total_chunks"`
	ReceivedChunks int      `json:"received_chunks"`
	FinalPath     string    `json:"final_path"`
	Size          int64     `json:"size"`
	Hash          string    `json:"hash"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
