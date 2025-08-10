package todo

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
)

// HTTPController handles requests to the REST API endpoints.
type HTTPController struct {
	logger *slog.Logger
	tasks  TaskRepository
}

// NewHTTPController creates an [HTTPController] with the given
// [TaskRepository].
func NewHTTPController(tasks TaskRepository) *HTTPController {
	return &HTTPController{
		logger: slog.Default(),
		tasks:  tasks,
	}
}

func (c *HTTPController) respond(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		c.logger.Warn("cannot write response", "cause", err)
	}
}

// CreateTask handles requests to create a new task.
func (c *HTTPController) CreateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Info("handling HTTP request", "method", r.Method, "endpoint", r.URL.Path)

	task, err := c.doCreateTask(r)
	if err != nil {
		c.logger.Warn("cannot create task", "cause", err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Info("created task", "id", task.ID, "summary", task.Summary)
	c.respond(w, http.StatusCreated, task)
}

func (c *HTTPController) doCreateTask(r *http.Request) (*taskDTO, *restError) {
	dto := &taskCreateDTO{}
	if err := json.NewDecoder(r.Body).Decode(dto); err != nil {
		return nil, newBadRequestError("invalid task", err)
	}
	taskCreate := newTaskCreateFromDTO(dto)
	task, err := c.tasks.Create(r.Context(), taskCreate)
	if err != nil {
		return nil, newInternalServerError("cannot create task", err)
	}
	return task.toDTO(), nil
}

// ListTasks handles the request to retrieve tasks.
func (c *HTTPController) ListTasks(w http.ResponseWriter, r *http.Request) {
	c.logger.Info("handling HTTP request", "method", r.Method, "endpoint", r.URL.Path)

	tasks, err := c.doListTasks(r)
	if err != nil {
		c.logger.Warn("cannot list tasks", "cause", err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Info("retrieved tasks", "count", len(tasks))
	c.respond(w, http.StatusOK, tasks)
}

func (c *HTTPController) doListTasks(r *http.Request) ([]taskDTO, *restError) {
	tasks, err := c.tasks.All(r.Context())
	if err != nil {
		return nil, newInternalServerError("cannot retrieve tasks", err)
	}
	return tasks.toDTOs(), nil
}

// UpdateTask handles requests to update an existing task.
func (c *HTTPController) UpdateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Info("handling HTTP request", "method", r.Method, "endpoint", r.URL.Path)

	task, err := c.doUpdateTask(r)
	if err != nil {
		c.logger.Warn("cannot update task", "cause", err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Info("updated task", "id", task.ID, "summary", task.Summary)
	c.respond(w, http.StatusOK, task)
}

func (c *HTTPController) doUpdateTask(r *http.Request) (*taskDTO, *restError) {
	id := r.PathValue("id")
	updateDTO := taskUpdateDTO{}
	if err := json.NewDecoder(r.Body).Decode(&updateDTO); err != nil {
		return nil, newBadRequestError("invalid task data", err)
	}
	update := newTaskUpdateFromDTO(updateDTO)
	task, err := c.tasks.Update(r.Context(), id, update)
	if err != nil {
		return nil, newInternalServerError("cannot update task", err)
	}
	return task.toDTO(), nil
}

// DeleteTask handles requests to delete an existing task.
func (c *HTTPController) DeleteTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Info("handling HTTP request", "method", r.Method, "endpoint", r.URL.Path)

	if err := c.doDeleteTask(r); err != nil {
		c.logger.Warn("cannot delete task", "cause", err)
		c.respond(w, err.status, err)
		return
	}

	c.respond(w, http.StatusNoContent, nil)
}

func (c *HTTPController) doDeleteTask(r *http.Request) *restError {
	err := c.tasks.Delete(r.Context(), r.PathValue("id"))
	if err != nil {
		if IsTaskNotFoundError(err) {
			return newNotFoundError("no such task", err)
		}
		return newInternalServerError("cannot delete task", err)
	}
	return nil
}

// GRPCController handles requests to the gRPC API endpoints.
type GRPCController struct {
	todopb.UnimplementedTodoServiceServer
	server ServerStatusProvider
	tasks  TaskRepository
}

// NewGRPCController creates a [GRPCController] with the given providers.
func NewGRPCController(server ServerStatusProvider, tasks TaskRepository) *GRPCController {
	return &GRPCController{
		server: server,
		tasks:  tasks,
	}
}

// Status handles gRPC requests to retrieve the server status.
func (c *GRPCController) Status(ctx context.Context, _ *todopb.StatusRequest) (*todopb.StatusResponse, error) {
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
func (c *GRPCController) CreateTask(
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
func (c *GRPCController) ListTasks(ctx context.Context, _ *todopb.ListTasksRequest) (*todopb.ListTasksResponse, error) {
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
func (c *GRPCController) UpdateTask(
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
func (c *GRPCController) DeleteTask(
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
