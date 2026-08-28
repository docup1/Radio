package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"radio/sender-service/internal/infrastructure/redis"
)

type Worker struct {
	svc *Service
	rdb *redis.Client
}

func NewWorker(svc *Service, rdb *redis.Client) *Worker {
	return &Worker{svc: svc, rdb: rdb}
}

func (w *Worker) Run(ctx context.Context) {
	go w.listenEvents(ctx)
	<-ctx.Done()
}

func (w *Worker) listenEvents(ctx context.Context) {
	w.recoverPending(ctx)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pollActiveStreams(ctx)
		}
	}
}

// recoverPending re-processes unacknowledged events from a previous run.
func (w *Worker) recoverPending(ctx context.Context) {
	activeStreams, err := w.rdb.GetActiveStreams(ctx)
	if err != nil {
		log.Printf("[worker] get active streams: %v", err)
		return
	}

	for _, streamID := range activeStreams {
		events, err := w.rdb.ReadPendingEvents(ctx, streamID)
		if err != nil {
			log.Printf("[worker] recover pending %s: %v", streamID, err)
			continue
		}
		for _, evt := range events {
			w.handleEvent(ctx, streamID, evt)
		}
	}
}

func (w *Worker) pollActiveStreams(ctx context.Context) {
	activeStreams, err := w.rdb.GetActiveStreams(ctx)
	if err != nil {
		log.Printf("[worker] get active streams: %v", err)
		return
	}

	for _, streamID := range activeStreams {
		if err := w.rdb.EnsureConsumerGroup(ctx, streamID); err != nil {
			log.Printf("[worker] ensure group %s: %v", streamID, err)
			continue
		}
		events, err := w.rdb.ReadEvents(ctx, streamID)
		if err != nil {
			log.Printf("[worker] read events %s: %v", streamID, err)
			continue
		}
		for _, evt := range events {
			w.handleEvent(ctx, streamID, evt)
		}
	}
}

type songPayload struct {
	ItemID string `json:"item_id"`
	SongID string `json:"song_id"`
}

func (w *Worker) handleEvent(ctx context.Context, streamID uuid.UUID, evt *redis.EventMessage) {
	switch evt.Type {
	case "stream_started", "song_changed":
		var payload songPayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			log.Printf("[worker] unmarshal %s: %v", evt.Type, err)
			break
		}
		songID, err := uuid.Parse(payload.SongID)
		if err != nil {
			log.Printf("[worker] parse song_id %q: %v", payload.SongID, err)
			break
		}
		log.Printf("[worker] %s stream=%s item=%s song=%s", evt.Type, streamID, payload.ItemID, songID)
		w.svc.StartHeartbeat(streamID)
		w.svc.OnSongChanged(streamID, songID)

	case "stream_stopped":
		log.Printf("[worker] stream_stopped stream=%s", streamID)
		w.svc.StopStream(streamID)

	case "queue_updated":
		log.Printf("[worker] queue_updated stream=%s", streamID)

	default:
		log.Printf("[worker] ignoring event type=%s stream=%s", evt.Type, streamID)
	}

	if err := w.rdb.AckEvent(ctx, streamID, evt.ID); err != nil {
		log.Printf("[worker] ack event %s: %v", evt.ID, err)
	}
}
