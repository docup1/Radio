package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"radio/stream-service/internal/infrastructure/redis"
)

type WorkerConfig struct {
	BatchSize        int
	OutboxInterval   time.Duration
	WatchdogInterval time.Duration
}

type Worker struct {
	svc          *Service
	pub          *redis.Publisher
	sub          *redis.Subscriber
	cfg          WorkerConfig
	signalCh     chan struct{}
}

func NewWorker(svc *Service, pub *redis.Publisher, sub *redis.Subscriber, cfg WorkerConfig) *Worker {
	return &Worker{
		svc:      svc,
		pub:      pub,
		sub:      sub,
		cfg:      cfg,
		signalCh: make(chan struct{}, 1),
	}
}

// Signal tells the worker to check outbox immediately (called by CRUD after commit).
func (w *Worker) Signal() {
	select {
	case w.signalCh <- struct{}{}:
	default:
	}
}

// Run starts the worker goroutines. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	go w.outboxPoller(ctx)
	go w.senderListener(ctx)
	go w.watchdog(ctx)
	<-ctx.Done()
}

// outboxPoller processes outbox events and publishes to Redis.
func (w *Worker) outboxPoller(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.OutboxInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.signalCh:
			w.processBatch(ctx)
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.svc.repos.Outbox.FetchUnprocessed(ctx, w.cfg.BatchSize)
	if err != nil {
		log.Printf("[worker] fetch outbox: %v", err)
		return
	}

	for _, event := range events {
		var payload map[string]interface{}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			payload = map[string]interface{}{}
		}

		// Get current revision from state
		st, err := w.svc.repos.State.GetByStreamID(ctx, event.AggregateID)
		if err != nil {
			log.Printf("[worker] get state for %s: %v", event.AggregateID, err)
			continue
		}

		redisEvent := redis.StreamEvent{
			Type:     event.EventType,
			StreamID: event.AggregateID,
			Revision: st.Revision,
			Payload:  payload,
		}

		if err := w.pub.Publish(ctx, redisEvent); err != nil {
			log.Printf("[worker] publish %s: %v", event.EventType, err)
			continue
		}
	}

	if len(events) > 0 {
		ids := make([]uuid.UUID, len(events))
		for i, e := range events {
			ids[i] = e.ID
		}
		if err := w.svc.repos.Outbox.MarkProcessed(ctx, ids); err != nil {
			log.Printf("[worker] mark processed: %v", err)
		}
	}
}

// senderListener handles SONG_COMPLETE events from the sender.
func (w *Worker) senderListener(ctx context.Context) {
	w.sub.ListenBlocks(ctx, func(event redis.SenderEvent) {
		log.Printf("[worker] sender event: stream=%s revision=%d", event.StreamID, event.Revision)
		if _, err := w.svc.AdvanceSong(ctx, event.StreamID); err != nil {
			log.Printf("[worker] advance song %s: %v", event.StreamID, err)
		}
	})
}

// watchdog checks active streams and auto-advances if song duration exceeded.
// Uses a simple heuristic: if started_at + 10 minutes < now, advance.
// TODO: query Content Service for actual song duration.
func (w *Worker) watchdog(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.WatchdogInterval)
	defer ticker.Stop()

	const maxSongDuration = 10 * time.Minute

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkWatchdog(ctx, maxSongDuration)
		}
	}
}

func (w *Worker) checkWatchdog(ctx context.Context, maxDuration time.Duration) {
	// For now, iterate active streams and check started_at
	// TODO: optimize with a dedicated query
	log.Printf("[worker] watchdog tick")
	// This will be implemented when we have a way to list active streams
}
