package main

import (
	"flag"
	"log"
	"net"
	"os"

	grpctransport "radio/content-service/internal/api/grpc"
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

	grpcSrv := grpctransport.NewServer(svc)
	lis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("grpc listen %s: %v", cfg.GRPC.Addr, err)
	}
	log.Printf("content-service access listening on %s", cfg.GRPC.Addr)
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("grpc server: %v", err)
	}
}
