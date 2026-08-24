package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"radio/sender-service/internal/domain/models"
)

const StreamEvents = "stream:events"
const ConsumerGroup = "sender-service"

type RedisSubscriber struct {
	rdb *redis.Client
}

func NewRedisSubscriber(rdb *redis.Client) *RedisSubscriber {
	// Ensure consumer group
	_ = rdb.XGroupCreateMkStream(context.Background(), StreamEvents, ConsumerGroup, "0").Err()
	return &RedisSubscriber{rdb: rdb}
}

type Worker struct {
	svc *Service
	sub *RedisSubscriber
}

func NewWorker(svc *Service, sub *RedisSubscriber) *Worker {
	return &Worker{svc: svc, sub: sub}
}

func (w *Worker) Run(ctx context.Context) {
	consumer := "sender-worker"

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := w.sub.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    ConsumerGroup,
			Consumer: consumer,
			Streams:  []string{StreamEvents, ">"},
			Count:    10,
			Block:    1 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			log.Printf("[worker] readgroup error: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range res {
			for _, msg := range stream.Messages {
				dataStr, ok := msg.Values["data"].(string)
				if !ok {
					continue
				}
				var event models.StreamEvent
				if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
					log.Printf("[worker] unmarshal: %v", err)
					continue
				}

				w.handleEvent(ctx, event)
			}
		}
	}
}

func (w *Worker) handleEvent(ctx context.Context, event models.StreamEvent) {
	streamID := event.StreamID

	switch event.Type {
	case "STREAM_STARTED":
		log.Printf("[worker] STREAM_STARTED stream=%s", streamID)
		w.svc.StartStream(streamID)

	case "STREAM_SONG_CHANGED":
		log.Printf("[worker] STREAM_SONG_CHANGED stream=%s revision=%d", streamID, event.Revision)
		w.svc.OnSongChanged(streamID)

	case "STREAM_ENDED":
		log.Printf("[worker] STREAM_ENDED stream=%s", streamID)
		w.svc.StopStream(streamID)

	case "QUEUE_UPDATED":
		log.Printf("[worker] QUEUE_UPDATED stream=%s", streamID)

	default:
		log.Printf("[worker] unknown event type: %s", event.Type)
	}
}
