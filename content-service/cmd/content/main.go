package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apirest "radio/content-service/internal/api/rest"
	grpctransport "radio/content-service/internal/api/grpc"
	httptransport "radio/content-service/internal/api/http"
	"radio/content-service/internal/application"
	"radio/content-service/internal/infrastructure"
	"radio/content-service/internal/infrastructure/db"
	osfs "radio/content-service/internal/infrastructure/os"
)

// main runs the content-service as a single binary/container. It starts three
// listeners in one process:
//   - REST API on HTTPPublic (public user-facing JSON),
//   - stream/audio API on HTTPPrivate (internal byte-range playback),
//   - gRPC Access.Check on GRPC (internal service-to-service).
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
	files := osfs.NewFileOpener()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	restSrv := &http.Server{
		Addr:              cfg.HTTPPublic.Addr,
		Handler:           apirest.NewHandler(svc, conn, cfg.Swagger.Enabled, cfg.Swagger.Path, cfg.Swagger.SpecFile),
		ReadHeaderTimeout: cfg.HTTPPublic.ReadHeaderTimeout,
	}
	streamSrv := &http.Server{
		Addr:              cfg.HTTPPrivate.Addr,
		Handler:           httptransport.NewHandler(svc, files, conn, cfg.Swagger.Enabled, cfg.Swagger.Path, cfg.Swagger.SpecFile),
		ReadHeaderTimeout: cfg.HTTPPrivate.ReadHeaderTimeout,
	}
	grpcSrv := grpctransport.NewServer(svc)

	errCh := make(chan error, 3)

	go func() {
		log.Printf("content-service REST listening on %s", cfg.HTTPPublic.Addr)
		if err := restSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		log.Printf("content-service stream listening on %s", cfg.HTTPPrivate.Addr)
		if err := streamSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	grpcLis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("grpc listen %s: %v", cfg.GRPC.Addr, err)
	}
	go func() {
		log.Printf("content-service gRPC listening on %s", cfg.GRPC.Addr)
		if err := grpcSrv.Serve(grpcLis); err != nil && err.Error() != "grpc: the server has been stopped" {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	case <-ctx.Done():
		log.Println("content-service shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = restSrv.Shutdown(shutdownCtx)
		_ = streamSrv.Shutdown(shutdownCtx)
		grpcSrv.GracefulStop()
	}
}
