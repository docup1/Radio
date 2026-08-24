package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type HashtagRepository struct {
	DB *sql.DB
}

func (r *HashtagRepository) Add(ctx context.Context, h *models.Hashtag) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO hashtags (id, stream_id, name) VALUES ($1, $2, $3)`,
		h.ID, h.StreamID, h.Name,
	)
	return err
}

func (r *HashtagRepository) ListByStream(ctx context.Context, streamID uuid.UUID) ([]*models.Hashtag, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, stream_id, name, created_at FROM hashtags WHERE stream_id = $1 ORDER BY created_at`, streamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hashtags []*models.Hashtag
	for rows.Next() {
		var h models.Hashtag
		if err := rows.Scan(&h.ID, &h.StreamID, &h.Name, &h.CreatedAt); err != nil {
			return nil, err
		}
		hashtags = append(hashtags, &h)
	}
	return hashtags, rows.Err()
}

func (r *HashtagRepository) Remove(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM hashtags WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *HashtagRepository) RemoveByStream(ctx context.Context, streamID uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM hashtags WHERE stream_id = $1`, streamID)
	return err
}
