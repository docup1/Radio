package application

import (
	"context"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type PlaylistSongService struct {
	PlaylistSongs interfaces.PlaylistSongRepository
	Playlists     interfaces.PlaylistRepository
}

type AddSongInput struct {
	SongID  uuid.UUID
	Position *int
}

func (s *PlaylistSongService) Add(ctx context.Context, playlistID, owner uuid.UUID, in AddSongInput) error {
	po, err := s.Playlists.OwnerOf(ctx, playlistID)
	if err != nil {
		return err
	}
	if po != owner {
		return interfaces.ErrForbidden
	}
	visible, err := s.PlaylistSongs.SongVisibleTo(ctx, in.SongID, owner)
	if err != nil {
		return err
	}
	if !visible {
		return interfaces.ErrNotFound
	}

	pos := 0
	if in.Position != nil {
		pos = *in.Position
	}
	if pos <= 0 {
		max, err := s.PlaylistSongs.MaxPosition(ctx, playlistID)
		if err != nil {
			return err
		}
		pos = max + 1
	}

	return s.PlaylistSongs.Add(ctx, models.PlaylistSong{
		PlaylistID: playlistID,
		SongID:     in.SongID,
		Position:   pos,
	})
}

func (s *PlaylistSongService) Remove(ctx context.Context, playlistID, owner, songID uuid.UUID) error {
	po, err := s.Playlists.OwnerOf(ctx, playlistID)
	if err != nil {
		return err
	}
	if po != owner {
		return interfaces.ErrForbidden
	}
	return s.PlaylistSongs.Remove(ctx, playlistID, songID)
}

func (s *PlaylistSongService) Move(ctx context.Context, playlistID, owner, songID uuid.UUID, position int) error {
	if position < 0 {
		return interfaces.ErrInvalid
	}
	po, err := s.Playlists.OwnerOf(ctx, playlistID)
	if err != nil {
		return err
	}
	if po != owner {
		return interfaces.ErrForbidden
	}
	return s.PlaylistSongs.Move(ctx, playlistID, songID, position)
}

// List returns the playlist's songs in playback order. The caller must own the
// playlist.
func (s *PlaylistSongService) List(ctx context.Context, playlistID, owner uuid.UUID) ([]models.Song, error) {
	po, err := s.Playlists.OwnerOf(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if po != owner {
		return nil, interfaces.ErrForbidden
	}
	return s.PlaylistSongs.ListSongs(ctx, playlistID)
}
