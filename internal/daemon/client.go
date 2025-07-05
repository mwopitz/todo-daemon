package daemon

import (
	"cmp"
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/mwopitz/todo-daemon/api/todopb"
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
func (c *Client) ServerStatus(ctx context.Context) (*pb.StatusResponse, error) {
	return c.daemon.Status(ctx, &pb.StatusRequest{})
}

func (c *Client) CreateTask(ctx context.Context, summary string) (*pb.Task, error) {
	task := &pb.NewTask{Summary: summary}
	resp, err := c.daemon.CreateTask(ctx, &pb.CreateTaskRequest{Task: task})
	if err != nil {
		return nil, fmt.Errorf("cannot create task: %w", err)
	}
	return resp.GetTask(), nil
}

// ListTasks retrieves the list of tasks from the To-do Daemon server.
func (c *Client) ListTasks(ctx context.Context) ([]*pb.Task, error) {
	resp, err := c.daemon.ListTasks(ctx, &pb.ListTasksRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetTasks(), nil
}

// CompleteTask marks the specified task as completed.
func (c *Client) CompleteTask(ctx context.Context, id string) (*pb.Task, error) {
	update := &pb.TaskUpdate{CompletedAt: timestamppb.Now()}
	fields, err := fieldmaskpb.New(update, "completed_at")
	if err != nil {
		return nil, err
	}
	req := &pb.UpdateTaskRequest{
		Id:     id,
		Update: update,
		Fields: fields,
	}
	res, err := c.daemon.UpdateTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.GetTask(), nil
}

func (c *Client) DeleteTask(ctx context.Context, id string) error {
	_, err := c.daemon.DeleteTask(ctx, &pb.DeleteTaskRequest{Id: id})
	if err != nil {
		return fmt.Errorf("cannot delete task: %w", err)
	}
	return nil
}
