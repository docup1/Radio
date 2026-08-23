package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type PlaylistRepository struct{ DB *sql.DB }

func (r *PlaylistRepository) Create(ctx context.Context, p models.Playlist) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO playlists (id, name, owner_id, is_public) VALUES ($1, $2, $3, $4)`,
		p.ID, p.Name, p.OwnerID, p.IsPublic)
	return err
}

func (r *PlaylistRepository) GetVisible(ctx context.Context, id, viewer uuid.UUID) (*models.Playlist, error) {
	p := &models.Playlist{}
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, name, owner_id, is_public, created_at, updated_at FROM playlists WHERE id = $1 AND (owner_id = $2 OR is_public = true)`,
		id, viewer).Scan(&p.ID, &p.Name, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return p, err
}

func (r *PlaylistRepository) ListVisible(ctx context.Context, viewer uuid.UUID, limit, offset int) ([]models.Playlist, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, name, owner_id, is_public, created_at, updated_at FROM playlists WHERE owner_id = $1 OR is_public = true
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, viewer, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Playlist, 0, limit)
	for rows.Next() {
		var p models.Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.OwnerID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PlaylistRepository) Update(ctx context.Context, id, owner uuid.UUID, patch interfaces.PlaylistPatch) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE playlists
		 SET name = COALESCE($1, name),
		     is_public = COALESCE($2, is_public),
		     updated_at = now()
		 WHERE id = $3 AND owner_id = $4`,
		patch.Name, patch.IsPublic, id, owner)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *PlaylistRepository) Delete(ctx context.Context, id, owner uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM playlists WHERE id = $1 AND owner_id = $2`, id, owner)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *PlaylistRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM playlists WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

func (r *PlaylistRepository) OwnerOf(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var owner uuid.UUID
	err := r.DB.QueryRowContext(ctx, `SELECT owner_id FROM playlists WHERE id = $1`, id).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, interfaces.ErrNotFound
	}
	return owner, err
}
