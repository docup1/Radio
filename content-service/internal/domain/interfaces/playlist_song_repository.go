package interfaces

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/models"
)

type PlaylistSongRepository interface {
	Add(ctx context.Context, item models.PlaylistSong) error
	Remove(ctx context.Context, playlistID, songID uuid.UUID) error
	List(ctx context.Context, playlistID uuid.UUID) ([]models.PlaylistSong, error)
	ListSongs(ctx context.Context, playlistID uuid.UUID) ([]models.Song, error)
	Move(ctx context.Context, playlistID, songID uuid.UUID, position int) error
	MaxPosition(ctx context.Context, playlistID uuid.UUID) (int, error)
	SongVisibleTo(ctx context.Context, songID, viewer uuid.UUID) (bool, error)
}
