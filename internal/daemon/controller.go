package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"math"

	pb "github.com/mwopitz/todo-daemon/internal/protogen"
)

type serverStatus struct {
	pid        int
	apiBaseURL string
}

type serverStatusProvider interface {
	status(ctx context.Context) (*serverStatus, error)
}

type serverStatusProviderFunc func(ctx context.Context) (*serverStatus, error)

func (f serverStatusProviderFunc) status(ctx context.Context) (*serverStatus, error) {
	return f(ctx)
}

// controller handles gRPC calls.
type controller struct {
	pb.UnimplementedTodoDaemonServer
	logger *log.Logger
	server serverStatusProvider
}

func newController(server serverStatusProvider, logger *log.Logger) *controller {
	return &controller{
		logger: cmp.Or(logger, log.Default()),
		server: server,
	}
}

func (c *controller) GetStatus(
	ctx context.Context,
	_ *pb.GetStatusRequest,
) (*pb.Status, error) {
	if c.server == nil {
		return nil, errors.New("cannot determine server status")
	}
	status, err := c.server.status(ctx)
	if err != nil {
		return nil, err
	}
	if status.pid < 0 || status.pid > math.MaxUint32 {
		return nil, fmt.Errorf("unexpected PID %d", status.pid)
	}
	pid := uint32(status.pid)
	return &pb.Status{
		Process: &pb.ProcessStatus{
			Pid: pid,
		},
		Server: &pb.ServerStatus{
			ApiBaseUrl: status.apiBaseURL,
		},
	}, nil
}
