package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type PlaylistSongRepository struct{ DB *sql.DB }

func (r *PlaylistSongRepository) Add(ctx context.Context, ps models.PlaylistSong) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO playlist_songs (playlist_id, song_id, position) VALUES ($1, $2, $3)`,
		ps.PlaylistID, ps.SongID, ps.Position)
	if err != nil {
		return pgError(err)
	}
	return nil
}

func (r *PlaylistSongRepository) Remove(ctx context.Context, playlistID, songID uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx,
		`DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2`, playlistID, songID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *PlaylistSongRepository) Move(ctx context.Context, playlistID, songID uuid.UUID, position int) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE playlist_songs SET position = $3 WHERE playlist_id = $1 AND song_id = $2`,
		playlistID, songID, position)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *PlaylistSongRepository) List(ctx context.Context, playlistID uuid.UUID) ([]models.PlaylistSong, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, playlist_id, song_id, position, created_at
		   FROM playlist_songs WHERE playlist_id = $1 ORDER BY position ASC`,
		playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.PlaylistSong, 0, 8)
	for rows.Next() {
		var ps models.PlaylistSong
		if err := rows.Scan(&ps.ID, &ps.PlaylistID, &ps.SongID, &ps.Position, &ps.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

// ListSongs returns the playlist's songs joined with the songs table, ordered by
// position. Used by the playlist detail view.
func (r *PlaylistSongRepository) ListSongs(ctx context.Context, playlistID uuid.UUID) ([]models.Song, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT s.id, s.name, s.description, s.owner_id, s.is_public, s.melody_id, s.image_id, s.created_at, s.updated_at
		   FROM playlist_songs ps JOIN songs s ON s.id = ps.song_id
		  WHERE ps.playlist_id = $1 ORDER BY ps.position ASC`,
		playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Song, 0, 8)
	for rows.Next() {
		var s models.Song
		var desc sql.NullString
		var img uuid.NullUUID
		if err := rows.Scan(songDests(&s, &desc, &img)...); err != nil {
			return nil, err
		}
		s.Description = desc.String
		s.ImageID = img
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PlaylistSongRepository) MaxPosition(ctx context.Context, playlistID uuid.UUID) (int, error) {
	var max sql.NullInt32
	err := r.DB.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), 0) FROM playlist_songs WHERE playlist_id = $1`, playlistID).
		Scan(&max)
	return int(max.Int32), err
}

func (r *PlaylistSongRepository) SongVisibleTo(ctx context.Context, songID, viewer uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM songs WHERE id = $1 AND (owner_id = $2 OR is_public = true))`,
		songID, viewer).Scan(&ok)
	return ok, err
}
