package todo

import (
	"cmp"
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/mwopitz/todo-daemon/api/todopb"
)

// HTTPController handles requests to the REST API endpoints.
type HTTPController struct {
	logger *log.Logger
	tasks  TaskRepository
}

// NewHTTPController creates an [HTTPController] with the given [TaskRepository]
// and an optional logger. If no logger is provided, the HTTP controller will
// use [log.Default].
func NewHTTPController(tasks TaskRepository, logger *log.Logger) *HTTPController {
	return &HTTPController{
		logger: cmp.Or(logger, log.Default()),
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
		c.logger.Printf("cannot write response: %v", err)
	}
}

// CreateTask handles requests to create a new task.
func (c *HTTPController) CreateTask(w http.ResponseWriter, r *http.Request) {
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	task, err := c.doCreateTask(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("created task %s: %s", task.ID, task.Summary)
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
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	tasks, err := c.doListTasks(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("retrieved %d tasks", len(tasks))
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
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	task, err := c.doUpdateTask(r)
	if err != nil {
		c.logger.Println(err)
		c.respond(w, err.status, err)
		return
	}

	c.logger.Printf("updated task %s: %s", task.ID, task.Summary)
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
	c.logger.Printf("handling HTTP request %s %s", r.Method, r.URL.Path)

	if err := c.doDeleteTask(r); err != nil {
		c.logger.Println(err)
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
	pb.UnimplementedTodoDaemonServer
	logger *log.Logger
	server ServerStatusProvider
	tasks  TaskRepository
}

// NewGRPCController creates a [GRPCController] with the given providers and an
// optional logger. If no logger is provided, it will use [log.Default].
func NewGRPCController(
	server ServerStatusProvider,
	tasks TaskRepository,
	logger *log.Logger,
) *GRPCController {
	return &GRPCController{
		logger: cmp.Or(logger, log.Default()),
		server: server,
		tasks:  tasks,
	}
}

// Status handles gRPC requests to retrieve the server status.
func (c *GRPCController) Status(ctx context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
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
	return &pb.StatusResponse{
		Pid:        uint32(pid),
		ApiBaseUrl: srv.APIBaseURL,
	}, nil
}

// CreateTask handles gRPC requests to create a new task in the to-do list.
func (c *GRPCController) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	task := newTaskCreateFromProto(req.GetTask())
	created, err := c.tasks.Create(ctx, task)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create task: %v", err)
	}
	return &pb.CreateTaskResponse{Task: created.toProto()}, nil
}

// ListTasks handles gRPC requests to retrieve tasks from the to-do list.
func (c *GRPCController) ListTasks(ctx context.Context, _ *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	if c.tasks == nil {
		return nil, status.Errorf(codes.Internal, "no task repository provided")
	}
	tasks, err := c.tasks.All(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot retrieve tasks: %v", err)
	}
	return &pb.ListTasksResponse{Tasks: tasks.toProtos()}, nil
}

// UpdateTask handles gRPC requests to update a task in the to-do list.
func (c *GRPCController) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
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
	return &pb.UpdateTaskResponse{Task: task.toProto()}, nil
}

// DeleteTask handles gRPC requests to delete a task from the to-do list.
func (c *GRPCController) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
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
	return &pb.DeleteTaskResponse{}, nil
}
