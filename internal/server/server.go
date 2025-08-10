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
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
	"github.com/mwopitz/todo-daemon/internal/todo"
)

func newInterceptorLoggerFunc(l *slog.Logger) logging.LoggerFunc {
	return func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	}
}

// Server implements the server of the To-do Daemon. It runs both an HTTP Server,
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

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(loggerFunc, loggingOpts...),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamServerInterceptor(loggerFunc, loggingOpts...),
		),
	)

	httpServer := &http.Server{
		Handler:           http.NewServeMux(),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		grpcServer: grpcServer,
		httpServer: httpServer,
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

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if err := todopb.RegisterTodoServiceHandlerFromEndpoint(
		ctx,
		mux,
		fmt.Sprintf("%s:%s", network, address),
		opts,
	); err != nil {
		return fmt.Errorf("cannot start gRPC gateway: %w", err)
	}
	s.httpServer.Handler.(*http.ServeMux).Handle("/api/", http.StripPrefix("/api", mux))

	grpcListener, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("cannot start gRPC server: %w", err)
	}

	grpcAddr := grpcListener.Addr().String()
	slog.Info("gRPC server listening on", "addr", grpcAddr)

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("cannot start HTTP server: %w", err)
	}

	httpAddr := httpListener.Addr().String()
	slog.Info("HTTP server listening on", "addr", httpAddr)

	status := func(_ context.Context) (*todo.ServerStatus, error) {
		u := url.URL{
			Scheme: "http",
			Host:   httpAddr,
			Path:   "/api",
		}
		return &todo.ServerStatus{
			PID:        os.Getpid(),
			APIBaseURL: u.String(),
		}, nil
	}

	// Connect the gRPC server to the controller.
	ctrl := todo.NewController(todo.ServerStatusProviderFunc(status), db)
	todopb.RegisterTodoServiceServer(s.grpcServer, ctrl)

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
