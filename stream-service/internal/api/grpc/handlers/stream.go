package handlers

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"radio/stream-service/internal/api/grpc/pb"
	"radio/stream-service/internal/application"
)

type StreamHandler struct {
	pb.UnimplementedStreamServiceServer
	svc *application.Service
}

func NewStreamHandler(svc *application.Service) *StreamHandler {
	return &StreamHandler{svc: svc}
}

func (h *StreamHandler) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.StreamStateResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stream_id")
	}

	st, err := h.svc.GetState(ctx, streamID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "stream state: %v", err)
	}

	resp := &pb.StreamStateResponse{
		StreamId:       st.StreamID.String(),
		IsActive:       st.IsActive,
		Revision:       st.Revision,
	}
	if st.CurrentQueueID != nil {
		resp.CurrentQueueId = st.CurrentQueueID.String()
		// Get song_id from queue item
		item, err := h.svc.GetQueueItem(ctx, *st.CurrentQueueID)
		if err == nil && item != nil {
			resp.SongId = item.SongID.String()
		}
	}
	if st.StartedAt != nil {
		resp.StartedAt = st.StartedAt.UnixNano()
	}
	return resp, nil
}

func (h *StreamHandler) Advance(ctx context.Context, req *pb.AdvanceRequest) (*pb.StreamStateResponse, error) {
	streamID, err := uuid.Parse(req.GetStreamId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stream_id")
	}

	st, err := h.svc.AdvanceSong(ctx, streamID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "advance: %v", err)
	}

	resp := &pb.StreamStateResponse{
		StreamId:       st.StreamID.String(),
		IsActive:       st.IsActive,
		Revision:       st.Revision,
	}
	if st.CurrentQueueID != nil {
		resp.CurrentQueueId = st.CurrentQueueID.String()
		item, err := h.svc.GetQueueItem(ctx, *st.CurrentQueueID)
		if err == nil && item != nil {
			resp.SongId = item.SongID.String()
		}
	}
	if st.StartedAt != nil {
		resp.StartedAt = st.StartedAt.UnixNano()
	}
	return resp, nil
}
