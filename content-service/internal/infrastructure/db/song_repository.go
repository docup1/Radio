package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type SongRepository struct{ DB *sql.DB }

func (r *SongRepository) Create(ctx context.Context, s models.Song) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO songs (id, name, description, owner_id, is_public, melody_id, image_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.Name, s.Description, s.OwnerID, s.IsPublic, s.MelodyID, s.ImageID)
	return err
}

func (r *SongRepository) GetVisible(ctx context.Context, id, viewer uuid.UUID) (*models.Song, error) {
	s := &models.Song{}
	var desc sql.NullString
	var img uuid.NullUUID
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, name, description, owner_id, is_public, melody_id, image_id, created_at, updated_at
		   FROM songs WHERE id = $1 AND (owner_id = $2 OR is_public = true)`,
		id, viewer).Scan(songDests(s, &desc, &img)...)
	s.Description = desc.String
	s.ImageID = img
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return s, err
}

func (r *SongRepository) ListVisible(ctx context.Context, viewer uuid.UUID, scope models.SongScope, limit, offset int) ([]models.Song, error) {
	var where string
	var args []any
	switch scope {
	case models.SongScopePublic:
		where = "WHERE is_public = true AND owner_id <> $1"
		args = []any{viewer, limit, offset}
	default:
		where = "WHERE owner_id = $1"
		args = []any{viewer, limit, offset}
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, name, description, owner_id, is_public, melody_id, image_id, created_at, updated_at
		   FROM songs `+where+`
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Song, 0, limit)
	for rows.Next() {
		s := models.Song{}
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

func (r *SongRepository) Update(ctx context.Context, id, owner uuid.UUID, patch interfaces.SongPatch) error {
	includeImage := patch.ImageID != nil
	res, err := r.DB.ExecContext(ctx,
		`UPDATE songs SET
			name = COALESCE($1, name),
			description = COALESCE($2, description),
			melody_id = COALESCE($3, melody_id),
			is_public = COALESCE($4, is_public),
			image_id = CASE WHEN $5 THEN $6 ELSE image_id END,
			updated_at = now()
		 WHERE id = $7 AND owner_id = $8`,
		patch.Name, patch.Description, patch.MelodyID, patch.IsPublic, includeImage, patch.ImageID, id, owner)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *SongRepository) Delete(ctx context.Context, id, owner uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM songs WHERE id = $1 AND owner_id = $2`, id, owner)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *SongRepository) SearchVisible(ctx context.Context, q string, viewer uuid.UUID, scope models.SongScope, limit, offset int) ([]models.Song, error) {
	ts := "(websearch_to_tsquery('russian', $1) || websearch_to_tsquery('english', $1))"
	var where string
	var args []any
	switch scope {
	case models.SongScopePublic:
		where = "WHERE search_vector @@ (" + ts + ") AND is_public = true AND owner_id <> $2"
		args = []any{q, viewer, limit, offset}
	default:
		where = "WHERE search_vector @@ (" + ts + ") AND owner_id = $2"
		args = []any{q, viewer, limit, offset}
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, name, description, owner_id, is_public, melody_id, image_id, created_at, updated_at
		   FROM songs `+where+`
		  ORDER BY ts_rank(search_vector, `+ts+`) DESC
		  LIMIT $3 OFFSET $4`, args...)
	if err != nil {
		return nil, pgError(err)
	}
	defer rows.Close()
	out := make([]models.Song, 0, limit)
	for rows.Next() {
		s := models.Song{}
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

func (r *SongRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM songs WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}

func (r *SongRepository) SongByMelody(ctx context.Context, melodyID uuid.UUID) (*models.Song, error) {
	s := &models.Song{}
	var desc sql.NullString
	var img uuid.NullUUID
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, name, description, owner_id, is_public, melody_id, image_id, created_at, updated_at
		   FROM songs WHERE melody_id = $1 LIMIT 1`, melodyID).
		Scan(songDests(s, &desc, &img)...)
	s.Description = desc.String
	s.ImageID = img
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return s, err
}
