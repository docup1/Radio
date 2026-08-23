package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type MelodyRepository struct{ DB *sql.DB }

func (r *MelodyRepository) Create(ctx context.Context, m models.Melody) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO melodies (id, path, content_type, size, hash) VALUES ($1, $2, $3, $4, $5)`,
		m.ID, m.Path, m.ContentType, m.Size, m.Hash)
	return err
}

func (r *MelodyRepository) Get(ctx context.Context, id uuid.UUID) (*models.Melody, error) {
	m := &models.Melody{}
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, path, content_type, size, hash, created_at FROM melodies WHERE id = $1`, id).
		Scan(&m.ID, &m.Path, &m.ContentType, &m.Size, &m.Hash, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return m, err
}

func (r *MelodyRepository) List(ctx context.Context, limit, offset int) ([]models.Melody, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, path, content_type, size, hash, created_at FROM melodies ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Melody, 0, limit)
	for rows.Next() {
		var m models.Melody
		if err := rows.Scan(&m.ID, &m.Path, &m.ContentType, &m.Size, &m.Hash, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MelodyRepository) Update(ctx context.Context, id uuid.UUID, patch interfaces.MelodyPatch) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE melodies SET
			path = COALESCE($1, path),
			content_type = COALESCE($2, content_type),
			size = COALESCE($3, size),
			hash = COALESCE($4, hash),
			updated_at = now()
		 WHERE id = $5`,
		patch.Path, patch.ContentType, patch.Size, patch.Hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *MelodyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM melodies WHERE id = $1`, id)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *MelodyRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM melodies WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}
