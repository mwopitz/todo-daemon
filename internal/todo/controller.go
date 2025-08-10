package todo

import (
	"context"
	"math"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
)

// Controller handles requests to the gRPC API endpoints.
type Controller struct {
	todopb.UnimplementedTodoServiceServer
	server ServerStatusProvider
	tasks  TaskRepository
}

// NewController creates a [Controller] with the given providers.
func NewController(server ServerStatusProvider, tasks TaskRepository) *Controller {
	return &Controller{
		server: server,
		tasks:  tasks,
	}
}

// Status handles gRPC requests to retrieve the server status.
func (c *Controller) Status(ctx context.Context, _ *todopb.StatusRequest) (*todopb.StatusResponse, error) {
	if c.server == nil {
		return nil, status.Errorf(codes.Internal, "no server status provided")
	}
	srv, err := c.server.Status(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot determine server status: %v", err)
	}
	pid := srv.PID
	if pid < 0 || pid > math.MaxUint32 {
		return nil, status.Errorf(codes.Internal, "invalid server PID: %d", pid)
	}
	return &todopb.StatusResponse{
		Pid:        uint32(pid),
		ApiBaseUrl: srv.APIBaseURL,
	}, nil
}

// CreateTask handles gRPC requests to create a new task in the to-do list.
func (c *Controller) CreateTask(
	ctx context.Context,
	req *todopb.CreateTaskRequest,
) (*todopb.CreateTaskResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	task := newTaskCreateFromProto(req.GetTask())
	created, err := c.tasks.Create(ctx, task)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create task: %v", err)
	}
	return &todopb.CreateTaskResponse{Task: created.toProto()}, nil
}

// ListTasks handles gRPC requests to retrieve tasks from the to-do list.
func (c *Controller) ListTasks(ctx context.Context, _ *todopb.ListTasksRequest) (*todopb.ListTasksResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	tasks, err := c.tasks.All(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot retrieve tasks: %v", err)
	}
	return &todopb.ListTasksResponse{Tasks: tasks.toProtos()}, nil
}

// UpdateTask handles gRPC requests to update a task in the to-do list.
func (c *Controller) UpdateTask(
	ctx context.Context,
	req *todopb.UpdateTaskRequest,
) (*todopb.UpdateTaskResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	id := req.GetId()
	update := newTaskUpdateFromProto(req.GetUpdate(), req.GetFields())
	task, err := c.tasks.Update(ctx, id, update)
	if err != nil {
		if IsTaskNotFoundError(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "cannot update task '%s': %v", id, err)
	}
	return &todopb.UpdateTaskResponse{Task: task.toProto()}, nil
}

// DeleteTask handles gRPC requests to delete a task from the to-do list.
func (c *Controller) DeleteTask(
	ctx context.Context,
	req *todopb.DeleteTaskRequest,
) (*todopb.DeleteTaskResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	id := req.GetId()
	if err := c.tasks.Delete(ctx, id); err != nil {
		if IsTaskNotFoundError(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "cannot delete task '%s': %v", id, err)
	}
	return &todopb.DeleteTaskResponse{}, nil
}
