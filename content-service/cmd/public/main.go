package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	apirest "radio/content-service/internal/api/rest"
	"radio/content-service/internal/application"
	"radio/content-service/internal/infrastructure"
	"radio/content-service/internal/infrastructure/db"
	osfs "radio/content-service/internal/infrastructure/os"

)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := infrastructure.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	conn, err := db.Connect(cfg.DB, cfg.DBPassword)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer conn.Close()

	if err := os.MkdirAll(cfg.Storage.ChunkDir, 0o755); err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := os.MkdirAll(cfg.Storage.FinalDir, 0o755); err != nil {
		log.Fatalf("storage: %v", err)
	}

	repos := application.Repos{
		Songs:          &db.SongRepository{DB: conn},
		Melodies:       &db.MelodyRepository{DB: conn},
		Images:         &db.ImageRepository{DB: conn},
		Playlists:      &db.PlaylistRepository{DB: conn},
		PlaylistSongs:  &db.PlaylistSongRepository{DB: conn},
		UploadSessions: &db.UploadSessionRepository{DB: conn},
	}
	svc := application.New(repos, application.StorageParams{
		ChunkStore:   osfs.NewChunkStore(cfg.Storage.ChunkDir, cfg.Storage.FinalDir),
		FinalRoot:    cfg.Storage.FinalDir,
		MaxChunkSize: cfg.Storage.MaxChunkSize,
		MaxFileSize:  cfg.Storage.MaxFileSize,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPPublic.Addr,
		Handler:           apirest.NewHandler(svc, conn, cfg.Swagger.Enabled, cfg.Swagger.Path, cfg.Swagger.SpecFile),
		ReadHeaderTimeout: cfg.HTTPPublic.ReadHeaderTimeout,
	}
	log.Printf("content-service public listening on %s", cfg.HTTPPublic.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
