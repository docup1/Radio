package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"radio/sender-service/internal/api"
	"radio/sender-service/internal/api/skip"
	"radio/sender-service/internal/api/websocket"
	"radio/sender-service/internal/application"
	"radio/sender-service/internal/infrastructure"
	contenthttp "radio/sender-service/internal/infrastructure/http"
	"radio/sender-service/internal/infrastructure/redis"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := infrastructure.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	hub := application.NewHub()
	contentClient := contenthttp.NewContentClient(cfg.ContentServiceURL, cfg.ChunkSize)

	rdb, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	svc := application.NewService(contentClient, rdb, hub, application.SenderConfig{
		ChunkSize:        cfg.ChunkSize,
		Bitrate:          cfg.Bitrate,
		BufferSeconds:    cfg.BufferSeconds,
		PrefetchCount:    cfg.PrefetchCount,
		NextSongPrefetch: cfg.NextSongPrefetch,
	})

	worker := application.NewWorker(svc, rdb)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go worker.Run(ctx)

	wsHandler := websocket.NewHandler(hub, svc)
	statusHandler := api.NewStatusHandler(hub)
	skipHandler := skip.NewHandler(svc)

	combined := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /stream/{id}/skip → skip handler
		if strings.HasPrefix(r.URL.Path, "/stream/") && strings.HasSuffix(r.URL.Path, "/skip") {
			skipHandler.ServeHTTP(w, r)
			return
		}
		// /stream/{id}/status → status handler
		if strings.HasPrefix(r.URL.Path, "/stream/") && strings.HasSuffix(r.URL.Path, "/status") {
			statusHandler.ServeHTTP(w, r)
			return
		}
		// /stream/{id} → WebSocket handler
		wsHandler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           combined,
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
