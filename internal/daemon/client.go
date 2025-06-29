package daemon

import (
	"cmp"
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/mwopitz/todo-daemon/internal/protogen"
)

// Client is used for communicating with the To-do Daemon server.
type Client struct {
	logger *log.Logger
	conn   *grpc.ClientConn
	daemon pb.TodoDaemonClient
}

// NewClient creates a To-do Daemon client and connects it to the server
// listening on the specified network address.
func NewClient(network, address string, logger *log.Logger) (*Client, error) {
	target := fmt.Sprintf("%s:%s", network, address)
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", target, err)
	}
	return &Client{
		logger: cmp.Or(logger, log.Default()),
		conn:   conn,
		daemon: pb.NewTodoDaemonClient(conn),
	}, nil
}

// Close closes the connection to the To-do Daemon server.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ServerStatus retrieves the address of the To-do Daemon server.
func (c *Client) ServerStatus(ctx context.Context) (*pb.Status, error) {
	return c.daemon.GetStatus(ctx, &pb.GetStatusRequest{})
}
