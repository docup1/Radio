package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"radio/sender-service/internal/application"
	"radio/sender-service/internal/infrastructure"
	"radio/sender-service/internal/infrastructure/grpc"
	contenthttp "radio/sender-service/internal/infrastructure/http"
	"radio/sender-service/internal/infrastructure/redis"
	"radio/sender-service/internal/api/websocket"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := infrastructure.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Connect to stream-service gRPC
	streamClient, err := grpc.NewStreamClient(cfg.StreamServiceGRPC)
	if err != nil {
		log.Fatalf("grpc connect: %v", err)
	}
	defer streamClient.Close()

	// Connect to Redis
	rdb, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	// Create hub (manages per-stream buffers + WebSocket listeners)
	hub := application.NewHub()

	// Content Service client (fetches audio chunks)
	contentClient := contenthttp.NewContentClient(cfg.ContentServiceURL, cfg.ChunkSize)

	// Create sender service
	svc := application.NewService(streamClient, contentClient, hub, application.SenderConfig{
		ChunkSize:     cfg.ChunkSize,
		BufferSeconds: cfg.BufferSeconds,
	})

	// Redis subscriber (listens to stream:events)
	sub := application.NewRedisSubscriber(rdb)

	// Worker (processes Redis events)
	worker := application.NewWorker(svc, sub)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start worker
	go worker.Run(ctx)

	// WebSocket handler
	wsHandler := websocket.NewHandler(hub, svc)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           wsHandler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("sender-service listening on %s", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	case <-ctx.Done():
		log.Println("sender-service shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}
