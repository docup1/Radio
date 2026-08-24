package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type StreamStateRepository struct {
	DB *sql.DB
}

func (r *StreamStateRepository) Create(ctx context.Context, st *models.StreamState) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO stream_state (stream_id, is_active, revision) VALUES ($1, $2, $3)`,
		st.StreamID, st.IsActive, st.Revision,
	)
	return err
}

func (r *StreamStateRepository) GetByStreamID(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT stream_id, current_queue_id, started_at, is_active, revision, updated_at
		 FROM stream_state WHERE stream_id = $1`, streamID,
	)
	var st models.StreamState
	if err := row.Scan(&st.StreamID, &st.CurrentQueueID, &st.StartedAt, &st.IsActive, &st.Revision, &st.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &st, nil
}

func (r *StreamStateRepository) Update(ctx context.Context, st *models.StreamState) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE stream_state
		 SET current_queue_id=$1, started_at=$2, is_active=$3, revision=$4, updated_at=now()
		 WHERE stream_id=$5`,
		st.CurrentQueueID, st.StartedAt, st.IsActive, st.Revision, st.StreamID,
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

// Advance atomically transitions to the next song: sets current_queue_id,
// started_at, bumps revision, and returns the updated state.
func (r *StreamStateRepository) Advance(ctx context.Context, streamID uuid.UUID, nextQueueID *uuid.UUID, startedAt time.Time) (*models.StreamState, error) {
	var st models.StreamState
	err := r.DB.QueryRowContext(ctx,
		`UPDATE stream_state
		 SET current_queue_id = $2, started_at = $3, revision = revision + 1, updated_at = now()
		 WHERE stream_id = $1
		 RETURNING stream_id, current_queue_id, started_at, is_active, revision, updated_at`,
		streamID, nextQueueID, startedAt,
	).Scan(&st.StreamID, &st.CurrentQueueID, &st.StartedAt, &st.IsActive, &st.Revision, &st.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &st, nil
}

// SetActive updates the is_active flag and bumps revision within a transaction.
func (r *StreamStateRepository) SetActive(ctx context.Context, streamID uuid.UUID, active bool) (*models.StreamState, error) {
	var st models.StreamState
	err := r.DB.QueryRowContext(ctx,
		`UPDATE stream_state
		 SET is_active = $2, revision = revision + 1, updated_at = now()
		 WHERE stream_id = $1
		 RETURNING stream_id, current_queue_id, started_at, is_active, revision, updated_at`,
		streamID, active,
	).Scan(&st.StreamID, &st.CurrentQueueID, &st.StartedAt, &st.IsActive, &st.Revision, &st.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, err
	}
	return &st, nil
}

// ListActive returns all streams that are currently active (is_active = true).
func (r *StreamStateRepository) ListActive(ctx context.Context) ([]*models.StreamState, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT stream_id, current_queue_id, started_at, is_active, revision, updated_at
		 FROM stream_state WHERE is_active = true`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*models.StreamState
	for rows.Next() {
		var st models.StreamState
		if err := rows.Scan(&st.StreamID, &st.CurrentQueueID, &st.StartedAt, &st.IsActive, &st.Revision, &st.UpdatedAt); err != nil {
			return nil, err
		}
		states = append(states, &st)
	}
	return states, rows.Err()
}
