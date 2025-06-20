package todo

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"math"
	"os"

	pb "github.com/mwopitz/go-daemon/internal/protogen"
)

type ServerStatus struct {
	PID int
}

type Controller struct {
	pb.UnimplementedTodoDaemonServer
	logger *log.Logger
	tasks *TaskService
}

func NewController(tasks *TaskService, logger *log.Logger) *Controller {
	return &Controller{
		logger: cmp.Or(logger, log.Default()),
		tasks: tasks,
	}
}

func (c *Controller) GetStatus(
	_ context.Context,
	_ *pb.GetStatusRequest,
) (*pb.GetStatusResponse, error) {
	pid := os.Getpid()
	if pid < 0 || pid > math.MaxUint32 {
		return nil, fmt.Errorf("invalid PID: %d", pid)
	}
	return &pb.GetStatusResponse{
		Process: &pb.Process{
			Pid: uint32(pid),
		},
	}, nil
}
