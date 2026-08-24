package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type OutboxRepository struct {
	DB *sql.DB
}

func (r *OutboxRepository) Insert(ctx context.Context, event *models.OutboxEvent) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload) VALUES ($1, $2, $3, $4, $5)`,
		event.ID, event.AggregateType, event.AggregateID, event.EventType, event.Payload,
	)
	return err
}

// FetchUnprocessed returns unprocessed outbox events in creation order.
// Uses FOR UPDATE SKIP LOCKED for safe concurrent consumption.
func (r *OutboxRepository) FetchUnprocessed(ctx context.Context, limit int) ([]*models.OutboxEvent, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, aggregate_type, aggregate_id, event_type, payload, created_at
		 FROM outbox
		 WHERE processed_at IS NULL
		 ORDER BY created_at
		 LIMIT $1
		 FOR UPDATE SKIP LOCKED`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.OutboxEvent
	for rows.Next() {
		var e models.OutboxEvent
		if err := rows.Scan(&e.ID, &e.AggregateType, &e.AggregateID, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, rows.Err()
}

func (r *OutboxRepository) MarkProcessed(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	// Build $1, $2, ... placeholder list
	query := `UPDATE outbox SET processed_at = now() WHERE id IN (`
	for i, id := range ids {
		if i > 0 {
			query += `,`
		}
		query += `'` + id.String() + `'`
	}
	query += `)`
	_, err := r.DB.ExecContext(ctx, query)
	return err
}
