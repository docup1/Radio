package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type QueueRepository struct {
	DB *sql.DB
}

func (r *QueueRepository) Add(ctx context.Context, item *models.StreamQueueItem) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO stream_queue (id, stream_id, song_id, position) VALUES ($1, $2, $3, $4)`,
		item.ID, item.StreamID, item.SongID, item.Position,
	)
	return err
}

func (r *QueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.StreamQueueItem, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, stream_id, song_id, position, created_at FROM stream_queue WHERE id = $1`, id,
	)
	var item models.StreamQueueItem
	if err := row.Scan(&item.ID, &item.StreamID, &item.SongID, &item.Position, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *QueueRepository) ListByStream(ctx context.Context, streamID uuid.UUID) ([]*models.StreamQueueItem, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, stream_id, song_id, position, created_at
		 FROM stream_queue WHERE stream_id = $1 ORDER BY position`, streamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.StreamQueueItem
	for rows.Next() {
		var item models.StreamQueueItem
		if err := rows.Scan(&item.ID, &item.StreamID, &item.SongID, &item.Position, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (r *QueueRepository) GetNextAfterPosition(ctx context.Context, streamID uuid.UUID, position int64) (*models.StreamQueueItem, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, stream_id, song_id, position, created_at
		 FROM stream_queue
		 WHERE stream_id = $1 AND position > $2
		 ORDER BY position LIMIT 1`, streamID, position,
	)
	var item models.StreamQueueItem
	if err := row.Scan(&item.ID, &item.StreamID, &item.SongID, &item.Position, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *QueueRepository) GetFirst(ctx context.Context, streamID uuid.UUID) (*models.StreamQueueItem, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, stream_id, song_id, position, created_at
		 FROM stream_queue WHERE stream_id = $1 ORDER BY position LIMIT 1`, streamID,
	)
	var item models.StreamQueueItem
	if err := row.Scan(&item.ID, &item.StreamID, &item.SongID, &item.Position, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *QueueRepository) GetByStreamAndPosition(ctx context.Context, streamID uuid.UUID, position int64) (*models.StreamQueueItem, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, stream_id, song_id, position, created_at
		 FROM stream_queue WHERE stream_id = $1 AND position = $2`, streamID, position,
	)
	var item models.StreamQueueItem
	if err := row.Scan(&item.ID, &item.StreamID, &item.SongID, &item.Position, &item.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *QueueRepository) Remove(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM stream_queue WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *QueueRepository) RemoveByStream(ctx context.Context, streamID uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM stream_queue WHERE stream_id = $1`, streamID)
	return err
}

func (r *QueueRepository) MaxPosition(ctx context.Context, streamID uuid.UUID) (int64, error) {
	var pos sql.NullInt64
	err := r.DB.QueryRowContext(ctx,
		`SELECT MAX(position) FROM stream_queue WHERE stream_id = $1`, streamID,
	).Scan(&pos)
	if err != nil {
		return 0, err
	}
	if !pos.Valid {
		return 0, nil
	}
	return pos.Int64, nil
}

func (r *QueueRepository) Count(ctx context.Context, streamID uuid.UUID) (int64, error) {
	var n int64
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stream_queue WHERE stream_id = $1`, streamID,
	).Scan(&n)
	return n, err
}

func (r *QueueRepository) Move(ctx context.Context, id uuid.UUID, newPosition int64) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE stream_queue SET position = $2 WHERE id = $1`, id, newPosition,
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
