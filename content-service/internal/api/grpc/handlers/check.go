package handlers

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"radio/content-service/internal/api/grpc/pb"
	"radio/content-service/internal/application"
)

// CheckHandler implements the gRPC AccessServer. Check reports whether the
// given user may access a song's audio: the song must be public or owned by the
// user, and its melody must exist.
type CheckHandler struct {
	pb.UnimplementedAccessServer
	svc *application.Services
}

func NewCheckHandler(svc *application.Services) *CheckHandler {
	return &CheckHandler{svc: svc}
}

func (h *CheckHandler) Check(ctx context.Context, req *pb.AccessCheckRequest) (*pb.AccessCheckResponse, error) {
	ownerID, err := uuid.Parse(req.GetOwnerId())
	if err != nil {
		return &pb.AccessCheckResponse{Allowed: false}, nil
	}
	songID, err := uuid.Parse(req.GetSongId())
	if err != nil {
		return &pb.AccessCheckResponse{Allowed: false}, nil
	}

	song, err := h.svc.Songs.Get(ctx, songID, ownerID)
	if err != nil {
		// Not found or not accessible to this user.
		return &pb.AccessCheckResponse{Allowed: false}, nil
	}
	exists, err := h.svc.Melodies.Exists(ctx, song.MelodyID)
	if err != nil {
		return nil, status.Error(codes.Internal, "melody check failed")
	}
	return &pb.AccessCheckResponse{Allowed: exists}, nil
}
