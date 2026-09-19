package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"radio/stream-service/internal/domain/models"
)

type StreamRepository struct {
	DB *sql.DB
}

func (r *StreamRepository) Create(ctx context.Context, s *models.Stream) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO streams (id, name, description) VALUES ($1, $2, $3)`,
		s.ID, s.Name, s.Description,
	)
	return err
}

func (r *StreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Stream, error) {
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM streams WHERE id = $1`, id,
	)
	s, err := scanStream(row)
	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	return s, err
}

// ListByIDs returns streams for the given IDs (order is not guaranteed).
func (r *StreamRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Stream, error) {
	if len(ids) == 0 {
		return []*models.Stream{}, nil
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, name, description, created_at, updated_at FROM streams WHERE id = ANY($1::uuid[])`,
		uuidArray(ids),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	streams := make([]*models.Stream, 0, len(ids))
	for rows.Next() {
		s, err := scanStream(rows)
		if err != nil {
			return nil, err
		}
		streams = append(streams, s)
	}
	return streams, rows.Err()
}

// SearchActive finds matching streams among the given active IDs using trigram
// similarity (pg_trgm) on name and description.
func (r *StreamRepository) SearchActive(ctx context.Context, ids []uuid.UUID, q string, limit, offset int) ([]*models.Stream, int, error) {
	if len(ids) == 0 {
		return []*models.Stream{}, 0, nil
	}
	q = strings.TrimSpace(q)
	query := `
		SELECT id, name, description, created_at, updated_at,
		       GREATEST(similarity(name, $2), COALESCE(similarity(description, $2), 0)) AS sim
		FROM streams
		WHERE id = ANY($1::uuid[])
		  AND (name % $2 OR description % $2 OR name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')
		ORDER BY sim DESC, updated_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.DB.QueryContext(ctx, query, uuidArray(ids), q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	streams := make([]*models.Stream, 0)
	for rows.Next() {
		s, err := scanStream(rows)
		if err != nil {
			return nil, 0, err
		}
		streams = append(streams, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.DB.QueryRowContext(ctx,
		`SELECT count(*) FROM streams
		 WHERE id = ANY($1::uuid[])
		   AND (name % $2 OR description % $2 OR name ILIKE '%'||$2||'%' OR description ILIKE '%'||$2||'%')`,
		uuidArray(ids), q,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	return streams, total, nil
}

func (r *StreamRepository) Update(ctx context.Context, s *models.Stream) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE streams SET name=$1, description=$2, updated_at=now() WHERE id=$3`,
		s.Name, s.Description, s.ID,
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStream(sc rowScanner) (*models.Stream, error) {
	var s models.Stream
	err := sc.Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// uuidArray renders UUIDs as a Postgres array literal usable with $1::uuid[].
func uuidArray(ids []uuid.UUID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}
