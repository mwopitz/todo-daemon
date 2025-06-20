package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"github.com/urfave/cli/v3"
)

// ErrAlreadyRunning is returned by [CLI.Run] when executing the run command
// while the Go Daemon server is already running.
var ErrAlreadyRunning = errors.New("another instance is already running")

// CLI implements the command-line interface of the Go Daemon.
type CLI struct {
	logger  *log.Logger
	rootCmd *cli.Command
}

// NewCLI creates a new CLI instance with the specified version and an optional
// logger. If no logger is provided, the CLI will use [log.Default] instead.
func NewCLI(version string, logger *log.Logger) *CLI {
	c := &CLI{}
	c.logger = cmp.Or(logger, log.Default())
	c.rootCmd = &cli.Command{
		Name:  "go-daemon",
		Usage: "A simple daemon server in Go",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the Go Daemon server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:      "lock",
						Usage:     "path to the lock file",
						Value:     defaultLockFile(),
						TakesFile: true,
					},
				},
				Action: c.runServer,
			},
			{
				Name:   "status",
				Usage:  "Print the status of the Go Daemon server",
				Action: c.printServerStatus,
			},
			{
				Name: "version",
				Usage: "Print the version of the Go Daemon",
				Action: c.printVersion,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "sock",
				Usage:     "path to the socket file",
				Value:     defaultSockFile(),
				TakesFile: true,
			},
		},
	}
	return c
}

// Run executes the CLI command specified by the given arguments.
func (c *CLI) Run(ctx context.Context, args []string) error {
	return c.rootCmd.Run(ctx, args)
}

func (c *CLI) runServer(ctx context.Context, cmd *cli.Command) error {
	lockFile := cmd.String("lock")
	sockFile := cmd.String("sock")

	err := os.MkdirAll(filepath.Dir(lockFile), 0700)
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	lock := flock.New(lockFile)
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	if !locked {
		return ErrAlreadyRunning
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			c.logger.Printf("cannot release file lock: %v", err)
		}
	}()
	c.logger.Printf("acquired file lock %s", lockFile)

	err = os.MkdirAll(filepath.Dir(sockFile), 0700)
	if err != nil {
		return fmt.Errorf("cannot create socket directory: %w", err)
	}
	if err := os.Remove(sockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove socket file: %w", err)
	}

	// Create the Go Daemon server and run it in a separate goroutine, so we can
	// wait for either the server to stop or ctx.Done is closed.
	srv := NewServer(c.logger)
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve("unix", sockFile)
		close(done)
	}()

	// Wait until either the server stops or the context gets canceled.
	select {
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			c.logger.Println(context.Cause(ctx))
		} else {
			c.logger.Println(err)
		}
		return srv.GracefulStop()
	case err := <-done:
		return err
	}
}

func (c *CLI) printServerStatus(ctx context.Context, cmd *cli.Command) error {
	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			c.logger.Printf("cannot close client: %v", err)
		}
	}()

	status, err := client.ServerStatus(ctx)
	if err != nil {
		return fmt.Errorf("cannot get status: %w", err)
	}
	if pid := status.GetProcess().GetPid(); pid > 0 {
		fmt.Printf("pid: %d\n", pid)
	}
	return nil
}

func (c *CLI) printVersion(_ context.Context, _ *cli.Command) error {
	_, err := fmt.Printf("go-daemon version %s\n", c.rootCmd.Version)
	return err
}
