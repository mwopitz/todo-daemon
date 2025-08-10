// Package client implements the gRPC client of the To-do Daemon.
package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
)

// Client is used for communicating with the To-do Daemon's gRPC server.
type Client struct {
	conn    *grpc.ClientConn
	service todopb.TodoServiceClient
}

// New creates a To-do Daemon client and connects it to the server listening on
// the specified network address.
func New(network, address string) (*Client, error) {
	target := fmt.Sprintf("%s:%s", network, address)
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", target, err)
	}
	return &Client{
		conn:    conn,
		service: todopb.NewTodoServiceClient(conn),
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
func (c *Client) ServerStatus(ctx context.Context) (*todopb.StatusResponse, error) {
	return c.service.Status(ctx, &todopb.StatusRequest{})
}

// CreateTask creates the specified task in the to-do list.
func (c *Client) CreateTask(ctx context.Context, summary string) (*todopb.Task, error) {
	task := &todopb.NewTask{Summary: summary}
	resp, err := c.service.CreateTask(ctx, &todopb.CreateTaskRequest{Task: task})
	if err != nil {
		return nil, fmt.Errorf("cannot create task: %w", err)
	}
	return resp.GetTask(), nil
}

// ListTasks retrieves the list of tasks from the To-do Daemon server.
func (c *Client) ListTasks(ctx context.Context) ([]*todopb.Task, error) {
	resp, err := c.service.ListTasks(ctx, &todopb.ListTasksRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetTasks(), nil
}

// CompleteTask marks the specified task as completed.
func (c *Client) CompleteTask(ctx context.Context, id string) (*todopb.Task, error) {
	update := &todopb.TaskUpdate{CompletedAt: timestamppb.Now()}
	fields, err := fieldmaskpb.New(update, "completed_at")
	if err != nil {
		return nil, err
	}
	req := &todopb.UpdateTaskRequest{
		Id:     id,
		Update: update,
		Fields: fields,
	}
	res, err := c.service.UpdateTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.GetTask(), nil
}

// DeleteTask removes the specified task from the to-do list.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	_, err := c.service.DeleteTask(ctx, &todopb.DeleteTaskRequest{Id: id})
	if err != nil {
		return fmt.Errorf("cannot delete task: %w", err)
	}
	return nil
}
