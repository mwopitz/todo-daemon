// Package server provides the server of the To-do Daemon.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"

	pb "github.com/mwopitz/todo-daemon/api/todopb"
	"github.com/mwopitz/todo-daemon/internal/todo"
)

func newInterceptorLoggerFunc(l *slog.Logger) logging.LoggerFunc {
	return func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	}
}

// Server implements the Server of the To-do Daemon. It runs both an HTTP Server,
// which provides a REST API to external applications, as well as a gRPC Server,
// which is used for internal communication between the To-do Daemon processes.
type Server struct {
	grpcServer *grpc.Server
	httpServer *http.Server
}

// New creates a new To-do Daemon server with an optional logger. If no
// logger is provided, it the server uses [slog.Default].
func New() *Server {
	logger := slog.Default()
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}
	loggerFunc := newInterceptorLoggerFunc(logger)
	return &Server{
		grpcServer: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				logging.UnaryServerInterceptor(loggerFunc, loggingOpts...),
			),
			grpc.ChainStreamInterceptor(
				logging.StreamServerInterceptor(loggerFunc, loggingOpts...),
			),
		),
		httpServer: &http.Server{
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 2 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
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

	slog.Info("gRPC server started", "addr", grpcListener.Addr())

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("cannot start HTTP server: %w", err)
	}

	httpAddr := httpListener.Addr()
	slog.Info("HTTP server started", "addr", httpAddr)

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
	ctrl := todo.NewHTTPController(tasks)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tasks", ctrl.ListTasks)
	mux.HandleFunc("POST /api/v1/tasks", ctrl.CreateTask)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", ctrl.UpdateTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", ctrl.DeleteTask)

	s.httpServer.Handler = mux
}

func (s *Server) initGRPCServer(server todo.ServerStatusProvider, tasks todo.TaskRepository) {
	ctrl := todo.NewGRPCController(server, tasks)
	pb.RegisterTodoDaemonServer(s.grpcServer, ctrl)
}

// StopGracefully stops both the HTTP server and the gRPC server. It waits until
// all active RPCs and HTTP requests are finished.
func (s *Server) StopGracefully() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
}
