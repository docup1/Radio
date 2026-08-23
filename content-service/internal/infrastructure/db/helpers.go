package db

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"radio/content-service/internal/domain/interfaces"
	"radio/content-service/internal/domain/models"
)

// pgError maps Postgres constraint errors to domain sentinels.
func pgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23503", "23505":
		return interfaces.ErrConflict
	case "42601":
		return interfaces.ErrInvalid
	default:
		return err
	}
}

func songDests(s *models.Song, desc *sql.NullString, img *uuid.NullUUID) []any {
	return []any{&s.ID, &s.Name, desc, &s.OwnerID, &s.IsPublic, &s.MelodyID, img, &s.CreatedAt, &s.UpdatedAt}
}
