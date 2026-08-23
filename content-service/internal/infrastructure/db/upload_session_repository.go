package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

type UploadSessionRepository struct{ DB *sql.DB }

func (r *UploadSessionRepository) Create(ctx context.Context, s models.UploadSession) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO upload_sessions
			(id, owner_id, media_type, status, content_type, total_chunks, received_chunks, final_path, size, hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		s.ID, s.OwnerID, s.MediaType, s.Status, s.ContentType,
		s.TotalChunks, s.ReceivedChunks, s.FinalPath, s.Size, s.Hash)
	return err
}

func (r *UploadSessionRepository) Get(ctx context.Context, id uuid.UUID) (*models.UploadSession, error) {
	s := &models.UploadSession{}
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, owner_id, media_type, status, content_type, total_chunks, received_chunks, final_path, size, hash, created_at, updated_at
		   FROM upload_sessions WHERE id = $1`, id).
		Scan(&s.ID, &s.OwnerID, &s.MediaType, &s.Status, &s.ContentType,
			&s.TotalChunks, &s.ReceivedChunks, &s.FinalPath, &s.Size, &s.Hash,
			&s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, interfaces.ErrNotFound
	}
	return s, err
}

func (r *UploadSessionRepository) Update(ctx context.Context, id uuid.UUID, patch interfaces.UploadSessionPatch) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE upload_sessions SET
			status = COALESCE($1, status),
			received_chunks = COALESCE($2, received_chunks),
			final_path = COALESCE($3, final_path),
			size = COALESCE($4, size),
			hash = COALESCE($5, hash),
			updated_at = now()
		 WHERE id = $6`,
		patch.Status, patch.ReceivedChunks, patch.FinalPath, patch.Size, patch.Hash, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *UploadSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM upload_sessions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return interfaces.ErrNotFound
	}
	return nil
}

func (r *UploadSessionRepository) ListByOwner(ctx context.Context, owner uuid.UUID, limit, offset int) ([]models.UploadSession, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, owner_id, media_type, status, content_type, total_chunks, received_chunks, final_path, size, hash, created_at, updated_at
		   FROM upload_sessions WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.UploadSession, 0, limit)
	for rows.Next() {
		var s models.UploadSession
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.MediaType, &s.Status, &s.ContentType,
			&s.TotalChunks, &s.ReceivedChunks, &s.FinalPath, &s.Size, &s.Hash,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
