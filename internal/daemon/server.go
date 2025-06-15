package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"

	pb "github.com/mwopitz/go-daemon/daemon"
)

type Server struct {
	pb.UnimplementedDaemonServer
	logger         *log.Logger
	grpcServer     *grpc.Server
	httpServer     *http.Server
	httpServerAddr string
}

// newServer creates a new go-daemon server with an optional logger.
//
// If no logger is provided, it defaults to [log.Default()].
func newServer(logger *log.Logger) *Server {
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

func (s *Server) Stop() error {
	if s.grpcServer != nil {
		s.grpcServer.Stop()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

func (s *Server) GracefulStop() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	return nil
}

// Status retrieves the status of the go-daemon server.
func (s *Server) Status(ctx context.Context, req *pb.StatusRequest) (*pb.StatusReply, error) {
	pid := int32(os.Getpid())
	apiBaseURL := fmt.Sprintf("http://%s/api", s.httpServerAddr)
	return &pb.StatusReply{
		Process: &pb.ServerProcess{
			Pid: &pid,
		},
		Urls: &pb.ServerUrls{
			ApiBaseUrl: &apiBaseURL,
		},
	}, nil
}
