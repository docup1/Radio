package grpc

import (
	grpcserver "google.golang.org/grpc"

	"radio/stream-service/internal/api/grpc/handlers"
	"radio/stream-service/internal/api/grpc/pb"
	"radio/stream-service/internal/application"
)

func NewServer(svc *application.Service) *grpcserver.Server {
	s := grpcserver.NewServer()
	pb.RegisterStreamServiceServer(s, handlers.NewStreamHandler(svc))
	return s
}
