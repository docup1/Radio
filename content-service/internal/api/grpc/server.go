package grpc

import (
	grpcserver "google.golang.org/grpc"

	"radio/content-service/internal/api/grpc/handlers"
	"radio/content-service/internal/api/grpc/pb"
	"radio/content-service/internal/application"
)

// NewServer builds the gRPC server exposing the Access.Check endpoint. It is
// run by the dedicated cmd/access binary on its own port, separate from the
// internal HTTP audio server (cmd/stream).
func NewServer(svc *application.Services) *grpcserver.Server {
	s := grpcserver.NewServer()
	pb.RegisterAccessServer(s, handlers.NewCheckHandler(svc))
	return s
}
