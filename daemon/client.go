package daemon

import (
	"cmp"
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	logger *log.Logger
}

// NewClient creates a client and connects it to the server running at the
// specified network and address.
func NewClient(network, address string, logger *log.Logger) (*Client, error) {
	target := fmt.Sprintf("%s:%s", network, address)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", target, err)
	}
	return &Client{
		conn:   conn,
		logger: cmp.Or(logger, log.Default()),
	}, nil
}

// Close closes the server connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ServerAddress retrieves the address of the server.
func (c *Client) ServerAddress(ctx context.Context) (*AddressReply, error) {
	client := NewDaemonClient(c.conn)
	return client.GetAddress(ctx, &AddressRequest{})
}
