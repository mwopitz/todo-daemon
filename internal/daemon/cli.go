package daemon

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofrs/flock"
	"github.com/urfave/cli/v3"
)

// CLI implements the command-line interface of the go-daemon.
type CLI struct {
	logger  *log.Logger
	rootCmd *cli.Command
}

// NewCLI creates a new CLI instance with an optional logger.
func NewCLI(logger *log.Logger) *CLI {
	c := &CLI{}
	c.logger = cmp.Or(logger, log.Default())
	c.rootCmd = &cli.Command{
		Name:  "go-daemon",
		Usage: "A simple daemon server in Go",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the go-daemon server",
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
				Usage:  "Get the status of the go-daemon server",
				Action: c.printServerStatus,
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
		return fmt.Errorf("cannot acquire file lock %s; already running?", lockFile)
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

	srv := newServer(c.logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, 1)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		errc <- srv.Serve("unix", sockFile)
		close(errc)
	}()

	select {
	case <-ctx.Done():
		c.logger.Print(ctx.Err())
		return srv.GracefulStop()
	case sig := <-sigc:
		c.logger.Print(sig)
		return srv.GracefulStop()
	case err := <-errc:
		return err
	}
}

func (c *CLI) printServerStatus(ctx context.Context, cmd *cli.Command) error {
	sockFile := cmd.String("sock")
	client, err := newClient("unix", sockFile, c.logger)
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
	if status.Process != nil {
		fmt.Printf("pid: %d\n", *status.Process.Pid)
	}
	if status.Urls != nil && status.Urls.ApiBaseUrl != nil {
		fmt.Printf("api_base_url: %s\n", *status.Urls.ApiBaseUrl)
	}
	return nil
}
