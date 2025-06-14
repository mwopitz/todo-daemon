package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"
)

type Server struct {
	UnimplementedDaemonServiceServer
	logger         *log.Logger
	grpcServer     *grpc.Server
	httpServer     *http.Server
	httpServerAddr string
}

func NewServer(logger *log.Logger) *Server {
	return &Server{
		logger: cmp.Or(logger, log.Default()),
	}
}

func (s *Server) Serve(network, address string) error {
	grpcListener, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("cannot start gRPC server: %w", err)
	}
	defer grpcListener.Close()

	s.logger.Printf("gRPC server listening on %s", grpcListener.Addr())

	httpListener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("cannot start HTTP server: %w", err)
	}
	defer httpListener.Close()

	s.logger.Printf("HTTP server listening on %s", httpListener.Addr())
	s.httpServerAddr = httpListener.Addr().String()

	s.grpcServer = grpc.NewServer()
	RegisterDaemonServiceServer(s.grpcServer, s)
	s.httpServer = &http.Server{}

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

func (s *Server) GetAddress(ctx context.Context, req *AddressRequest) (*AddressReply, error) {
	return &AddressReply{
		Network: "tcp",
		Address: s.httpServerAddr,
	}, nil
}
