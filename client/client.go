package client

import (
	"context"
	"fmt"

	pb "github.com/mwopitz/go-daemon/todo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
}

// New creates a client and connects it to the specified todo server.
func New(network, address string) (*Client, error) {
	target := fmt.Sprintf("%s:%s", network, address)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("go-daemon: cannot connect to %s: %w", target, err)
	}
	return &Client{conn: conn}, nil
}

// Close closes the todo server connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ServerAddress retrieves the address of the go-daemon server.
func (c *Client) ServerAddress(ctx context.Context) (*pb.AddressReply, error) {
	client := pb.NewTodoServiceClient(c.conn)
	return client.GetAddress(ctx, &pb.AddressRequest{})
}
