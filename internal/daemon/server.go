package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	pb "github.com/mwopitz/todo-daemon/api/todopb"
	"github.com/mwopitz/todo-daemon/internal/todo"
	"google.golang.org/grpc"
)

// Server implements the server of the To-do Daemon. It runs both an HTTP server,
// which provides a REST API to external applications, as well as a gRPC server,
// which is used for internal communication between the To-do Daemon processes.
type Server struct {
	logger     *log.Logger
	grpcServer *grpc.Server
	httpServer *http.Server
}

// NewServer creates a new To-do Daemon server with an optional logger. If no
// logger is provided, it the server uses [log.Default].
func NewServer(logger *log.Logger) *Server {
	s := &Server{
		logger:     cmp.Or(logger, log.Default()),
		grpcServer: grpc.NewServer(),
		httpServer: &http.Server{},
	}
	return s
}

// Serve starts both the underlying HTTP server and gRPC server. The specified
// network and address arguments are only used for the gRPC server; the HTTP
// server always listens on IPv4 localhost + a random free port.
func (s *Server) Serve(network, address string) error {
	db := todo.NewInMemoryTaskDB()
	// Add some demo data...
	tasks := []todo.TaskCreate{
		{Summary: "Get some milk 🥛"},
		{Summary: "Walk the dog 🐕"},
		{Summary: "Take over the world! 🌍"},
	}
	ctx := context.Background()
	for _, task := range tasks {
		if _, err := db.Create(ctx, &task); err != nil {
			return err
		}
	}

	grpcListener, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("cannot start gRPC server: %w", err)
	}

	s.logger.Printf("gRPC server listening on %s", grpcListener.Addr())

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("cannot start HTTP server: %w", err)
	}

	httpAddr := httpListener.Addr()
	s.logger.Printf("HTTP server listening on %s", httpAddr)

	status := func(_ context.Context) (*todo.ServerStatus, error) {
		u := url.URL{
			Scheme: "http",
			Host:   httpAddr.String(),
			Path:   "/api",
		}
		return &todo.ServerStatus{
			PID:        os.Getpid(),
			APIBaseURL: u.String(),
		}, nil
	}

	s.initHTTPServer(db)
	s.initGRPCServer(todo.ServerStatusProviderFunc(status), db)

	grpcDone := make(chan error, 1)
	go func() {
		grpcDone <- s.grpcServer.Serve(grpcListener)
		close(grpcDone)
	}()

	httpDone := make(chan error, 1)
	go func() {
		httpDone <- s.httpServer.Serve(httpListener)
		close(httpDone)
	}()

	return errors.Join(<-grpcDone, <-httpDone)
}

func (s *Server) initHTTPServer(tasks todo.TaskRepository) {
	ctrl := todo.NewHTTPController(tasks, s.logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tasks", ctrl.GetTasks)
	mux.HandleFunc("POST /api/v1/tasks", ctrl.CreateTask)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", ctrl.UpdateTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", ctrl.DeleteTask)

	s.httpServer.Handler = mux
}

func (s *Server) initGRPCServer(server todo.ServerStatusProvider, tasks todo.TaskRepository) {
	ctrl := todo.NewGRPCController(server, tasks, s.logger)
	pb.RegisterTodoDaemonServer(s.grpcServer, ctrl)
}

// Stop stops both the HTTP server and the gRPC server immediately. It does not
// wait for active RPCs or HTTP requests to complete.
func (s *Server) Stop() error {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// GracefulStop stops both the HTTP server and the gRPC server. It waits until
// all active RPCs and HTTP requests are finished.
func (s *Server) GracefulStop() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
}
