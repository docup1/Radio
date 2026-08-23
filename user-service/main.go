package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"

	"radio/user-service/handlers"
	"radio/user-service/infra"
	"radio/user-service/jobs"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := infra.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := sql.Open("pgx", cfg.DB.DSN(cfg.DBPassword))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(context.Background(), cfg.DB.PingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	hasher := infra.NewHasher(cfg.Bcrypt.MaxConcurrent)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /register", handlers.RegisterHandler(db, cfg, hasher))
	mux.HandleFunc("POST /login", handlers.LoginHandler(db, cfg, hasher))
	mux.HandleFunc("GET /healthz", handlers.HealthHandler(db))
	mux.Handle("GET /me", infra.RequireAuth(db, cfg, handlers.MeHandler(db)))
	mux.Handle("POST /logout", infra.RequireAuth(db, cfg, handlers.LogoutHandler(db)))
	mux.Handle("PUT /password", infra.RequireAuth(db, cfg, handlers.PasswordHandler(db, cfg, hasher)))

	stopSessionCleaner := jobs.StartSessionCleaner(db, cfg.Auth.CleanupInterval)
	defer stopSessionCleaner()

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}
	log.Printf("user-service listening on %s", cfg.HTTP.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
