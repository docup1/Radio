package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"radio/stream-service/internal/api/rest"
	"radio/stream-service/internal/application"
	"radio/stream-service/internal/infrastructure"
	"radio/stream-service/internal/infrastructure/db"
	contentgrpc "radio/stream-service/internal/infrastructure/grpc/content"
	svcredis "radio/stream-service/internal/infrastructure/redis"
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

	rdb, err := svcredis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	pub := svcredis.NewPublisher(rdb)
	queueStore := svcredis.NewQueueStore(rdb)

	checkCache := contentgrpc.NewCheckCache(5 * time.Minute)
	contentClient, err := contentgrpc.NewContentClient(cfg.ContentServiceGRPC, checkCache)
	if err != nil {
		log.Fatalf("content grpc connect: %v", err)
	}
	defer contentClient.Close()

	repos := application.Repos{
		Streams:  &db.StreamRepository{DB: conn},
		Hashtags: &db.HashtagRepository{DB: conn},
	}
	svc := application.New(repos, queueStore, pub, contentClient, rdb)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           rest.NewHandler(svc),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("stream-service listening on %s", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	case <-ctx.Done():
		log.Println("stream-service shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}
