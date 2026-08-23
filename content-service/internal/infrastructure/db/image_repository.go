package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type ImageRepository struct{ DB *sql.DB }

func (r *ImageRepository) Create(ctx context.Context, m models.Image) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO images (id, path, content_type) VALUES ($1, $2, $3)`,
		m.ID, m.Path, m.ContentType)
	return err
}

func (r *ImageRepository) Get(ctx context.Context, id uuid.UUID) (*models.Image, error) {
	m := &models.Image{}
	err := r.DB.QueryRowContext(ctx, `SELECT id, path, content_type, created_at FROM images WHERE id = $1`, id).
		Scan(&m.ID, &m.Path, &m.ContentType, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return m, err
}

func (r *ImageRepository) List(ctx context.Context, limit, offset int) ([]models.Image, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, path, content_type, created_at FROM images ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Image, 0, limit)
	for rows.Next() {
		var m models.Image
		if err := rows.Scan(&m.ID, &m.Path, &m.ContentType, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ImageRepository) Update(ctx context.Context, id uuid.UUID, patch interfaces.ImagePatch) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE images SET path = COALESCE($1, path), content_type = COALESCE($2, content_type) WHERE id = $3`,
		patch.Path, patch.ContentType, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *ImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM images WHERE id = $1`, id)
	if err != nil {
		return pgError(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *ImageRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var ok bool
	err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM images WHERE id = $1)`, id).Scan(&ok)
	return ok, err
}
