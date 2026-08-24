package models

import (
	"time"

	"github.com/google/uuid"
)

const HashtagNameMaxLength = 64

type Hashtag struct {
	ID        uuid.UUID `json:"id"`
	StreamID  uuid.UUID `json:"stream_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
