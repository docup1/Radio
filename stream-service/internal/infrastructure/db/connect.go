package db

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"radio/stream-service/internal/infrastructure"
)

func Connect(cfg infrastructure.DBConfig, password string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", cfg.DSN(password))
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}
	return conn, nil
}
