package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	grpctransport "radio/stream-service/internal/api/grpc"
	"radio/stream-service/internal/api/rest"
	"radio/stream-service/internal/application"
	"radio/stream-service/internal/infrastructure"
	"radio/stream-service/internal/infrastructure/db"
	"radio/stream-service/internal/infrastructure/redis"
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

	rdb, err := redis.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}

	pub := redis.NewPublisher(rdb)
	sub := redis.NewSubscriber(rdb)

	repos := application.Repos{
		Streams:  &db.StreamRepository{DB: conn},
		State:    &db.StreamStateRepository{DB: conn},
		Queue:    &db.QueueRepository{DB: conn},
		Hashtags: &db.HashtagRepository{DB: conn},
		Outbox:   &db.OutboxRepository{DB: conn},
	}
	svc := application.New(repos, pub)

	worker := application.NewWorker(svc, pub, sub, application.WorkerConfig{
		BatchSize:        cfg.Worker.OutboxBatchSize,
		OutboxInterval:   cfg.Worker.OutboxInterval,
		WatchdogInterval: cfg.Worker.WatchdogInterval,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go worker.Run(ctx)

	// HTTP server
	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           rest.NewHandler(svc, worker),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
	}

	// gRPC server
	grpcSrv := grpctransport.NewServer(svc)
	grpcLis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatalf("grpc listen %s: %v", cfg.GRPC.Addr, err)
	}

	errCh := make(chan error, 2)

	go func() {
		log.Printf("stream-service REST listening on %s", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		log.Printf("stream-service gRPC listening on %s", cfg.GRPC.Addr)
		if err := grpcSrv.Serve(grpcLis); err != nil && err.Error() != "grpc: the server has been stopped" {
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
		grpcSrv.GracefulStop()
	}
}
