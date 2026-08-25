package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"radio/sender-service/internal/domain/models"
	pb "radio/sender-service/internal/infrastructure/grpc/pb"
)

type StreamClient struct {
	conn   *grpc.ClientConn
	client pb.StreamServiceClient
}

func NewStreamClient(addr string) (*StreamClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	return &StreamClient{
		conn:   conn,
		client: pb.NewStreamServiceClient(conn),
	}, nil
}

func (c *StreamClient) Close() error {
	return c.conn.Close()
}

func (c *StreamClient) GetState(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	resp, err := c.client.GetState(ctx, &pb.GetStateRequest{
		StreamId: streamID.String(),
	})
	if err != nil {
		return nil, err
	}

	st := &models.StreamState{
		StreamID:  streamID,
		IsActive:  resp.GetIsActive(),
		Revision:  resp.GetRevision(),
	}

	if resp.GetCurrentQueueId() != "" {
		id, err := uuid.Parse(resp.GetCurrentQueueId())
		if err == nil {
			st.CurrentQueueID = &id
		}
	}
	if resp.GetSongId() != "" {
		id, err := uuid.Parse(resp.GetSongId())
		if err == nil {
			st.SongID = &id
		}
	}
	if resp.GetStartedAt() != 0 {
		t := nanosToTime(resp.GetStartedAt())
		st.StartedAt = &t
	}

	return st, nil
}

func (c *StreamClient) Advance(ctx context.Context, streamID uuid.UUID) (*models.StreamState, error) {
	resp, err := c.client.Advance(ctx, &pb.AdvanceRequest{
		StreamId: streamID.String(),
	})
	if err != nil {
		return nil, err
	}

	st := &models.StreamState{
		StreamID:  streamID,
		IsActive:  resp.GetIsActive(),
		Revision:  resp.GetRevision(),
	}

	if resp.GetCurrentQueueId() != "" {
		id, err := uuid.Parse(resp.GetCurrentQueueId())
		if err == nil {
			st.CurrentQueueID = &id
		}
	}
	if resp.GetSongId() != "" {
		id, err := uuid.Parse(resp.GetSongId())
		if err == nil {
			st.SongID = &id
		}
	}
	if resp.GetStartedAt() != 0 {
		t := nanosToTime(resp.GetStartedAt())
		st.StartedAt = &t
	}

	return st, nil
}

func nanosToTime(nanos int64) time.Time {
	return time.Unix(0, nanos)
}
