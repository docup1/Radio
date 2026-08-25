package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type StreamRepository struct {
	DB *sql.DB
}

func (r *StreamRepository) Create(ctx context.Context, s *models.Stream) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO streams (id, name, description, loop) VALUES ($1, $2, $3, $4)`,
		s.ID, s.Name, s.Description, s.Loop,
	)
	return err
}

func (r *StreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Stream, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, name, description, loop, created_at, updated_at FROM streams WHERE id = $1`, id,
	)
	var s models.Stream
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Loop, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *StreamRepository) Update(ctx context.Context, s *models.Stream) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE streams SET name=$1, description=$2, loop=$3, updated_at=now() WHERE id=$4`,
		s.Name, s.Description, s.Loop, s.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *StreamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM streams WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}
