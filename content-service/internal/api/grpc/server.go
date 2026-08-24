package grpc

import (
	grpcserver "google.golang.org/grpc"

	"radio/content-service/internal/api/grpc/handlers"
	"radio/content-service/internal/api/grpc/pb"
	"radio/content-service/internal/application"
)

// NewServer builds the gRPC server exposing the Access.Check endpoint. It runs
// as the third listener inside the single content-service binary, alongside the
// REST (HTTPPublic) and stream/audio (HTTPPrivate) HTTP servers.
func NewServer(svc *application.Services) *grpcserver.Server {
	s := grpcserver.NewServer()
	pb.RegisterAccessServer(s, handlers.NewCheckHandler(svc))
	return s
}
