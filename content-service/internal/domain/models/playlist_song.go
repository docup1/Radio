package models

import (
	"time"

	"github.com/google/uuid"
)

type PlaylistSong struct {
	ID         uuid.UUID `json:"id"`
	PlaylistID uuid.UUID `json:"playlist_id"`
	SongID     uuid.UUID `json:"song_id"`
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}
