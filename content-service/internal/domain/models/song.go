package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	SongNameMaxLength        = 128
	SongDescriptionMaxLength = 512
)

// SongScope selects whose songs a listing/search query returns.
type SongScope string

const (
	// SongScopeMine returns only the caller's own songs.
	SongScopeMine SongScope = "mine"
	// SongScopePublic returns other users' public songs (owner_id != caller).
	SongScopePublic SongScope = "public"
)

// ParseSongScope normalizes a raw query value, defaulting to SongScopeMine.
func ParseSongScope(s string) SongScope {
	if SongScope(s) == SongScopePublic {
		return SongScopePublic
	}
	return SongScopeMine
}

type Song struct {
	ID          uuid.UUID      `json:"id"`
	OwnerID     uuid.UUID      `json:"owner_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	MelodyID    uuid.UUID      `json:"melody_id"`
	ImageID     uuid.NullUUID  `json:"image_id" swaggerignore:"true"`
	IsPublic    bool           `json:"is_public"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
