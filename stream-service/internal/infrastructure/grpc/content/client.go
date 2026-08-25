package content

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "radio/stream-service/internal/infrastructure/grpc/content/pb"
)

type ContentClient struct {
	conn   *grpc.ClientConn
	client pb.AccessClient
	cache  *CheckCache
}

func NewContentClient(addr string, cache *CheckCache) (*ContentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial content-service: %w", err)
	}
	return &ContentClient{
		conn:   conn,
		client: pb.NewAccessClient(conn),
		cache:  cache,
	}, nil
}

func (c *ContentClient) Close() error {
	return c.conn.Close()
}

func (c *ContentClient) Check(ctx context.Context, ownerID, songID string) (bool, error) {
	if allowed, ok := c.cache.Get(ownerID, songID); ok {
		return allowed, nil
	}

	resp, err := c.client.Check(ctx, &pb.AccessCheckRequest{
		OwnerId: ownerID,
		SongId:  songID,
	})
	if err != nil {
		return false, fmt.Errorf("content Check: %w", err)
	}

	c.cache.Set(ownerID, songID, resp.Allowed)
	return resp.Allowed, nil
}
