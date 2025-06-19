package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"

	pb "github.com/mwopitz/go-daemon/daemon"
)

// Server implements the server of the Go Daemon. It runs both an HTTP server,
// which provides a REST API to external applications, as well as a gRPC server,
// which is used for internal communication between the Go Daemon processes.
type Server struct {
	pb.UnimplementedDaemonServer
	logger         *log.Logger
	grpcServer     *grpc.Server
	httpServer     *http.Server
	httpServerAddr string
}

// NewServer creates a new Go Daemon server with an optional logger. If no
// logger is provided, it the server uses [log.Default].
func NewServer(logger *log.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{
		logger:     cmp.Or(logger, log.Default()),
		grpcServer: grpc.NewServer(),
		httpServer: &http.Server{
			Handler: mux,
		},
	}
	pb.RegisterDaemonServer(s.grpcServer, s)
	return s
}

// Serve starts both the underlying HTTP server and gRPC server. The specified
// network and address arguments are only used for the gRPC server; the HTTP
// server always listens on IPv4 localhost + a random free port.
func (s *Server) Serve(network, address string) error {
	grpcListener, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("cannot start gRPC server: %w", err)
	}

	s.logger.Printf("gRPC server listening on %s", grpcListener.Addr())

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("cannot start HTTP server: %w", err)
	}

	s.logger.Printf("HTTP server listening on %s", httpListener.Addr())
	s.httpServerAddr = httpListener.Addr().String()

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

// Status retrieves the status of the Go Daemon server.
func (s *Server) Status(
	_ context.Context,
	_ *pb.StatusRequest,
) (*pb.StatusReply, error) {
	pid := os.Getpid()
	if pid < 0 || pid > math.MaxUint32 {
		return nil, fmt.Errorf("invalid PID: %d", pid)
	}
	pidu := uint32(pid)
	apiBaseURL := fmt.Sprintf("http://%s/api", s.httpServerAddr)
	return &pb.StatusReply{
		Process: &pb.ServerProcess{
			Pid: &pidu,
		},
		Urls: &pb.ServerUrls{
			ApiBaseUrl: &apiBaseURL,
		},
	}, nil
}
