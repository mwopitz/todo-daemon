package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofrs/flock"
	"github.com/mwopitz/go-daemon/daemon"
	"github.com/urfave/cli/v3"
)

// CLI implements the command-line interface of the go-daemon.
type CLI struct {
	logger  *log.Logger
	rootCmd *cli.Command
}

func New(logger *log.Logger) *CLI {
	c := new(CLI)
	if logger != nil {
		c.logger = logger
	} else {
		c.logger = log.New(os.Stderr, "go-daemon: ", log.Lmsgprefix)
	}
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
				Name:   "address",
				Usage:  "Get the address of the go-daemon server",
				Action: c.printServerAddress,
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

// Run executes the CLI with the provided context and arguments.
func (c *CLI) Run(ctx context.Context, args []string) error {
	return c.rootCmd.Run(ctx, args)
}

func (c *CLI) runServer(ctx context.Context, cmd *cli.Command) error {
	lockFile := cmd.String("lock")
	sockFile := cmd.String("sock")

	err := os.MkdirAll(filepath.Dir(lockFile), 0755)
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
	defer lock.Unlock()

	if err := os.Remove(sockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove socket file: %w", err)
	}

	srv := daemon.NewServer()
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
		log.Printf("go-daemon: %v", ctx.Err())
		return srv.GracefulStop()
	case sig := <-sigc:
		log.Printf("go-daemon: %s", sig)
		return srv.GracefulStop()
	case err := <-errc:
		return err
	}
}

func (c *CLI) printServerAddress(ctx context.Context, cmd *cli.Command) error {
	sockFile := cmd.String("sockfile")
	client, err := daemon.NewClient("unix", sockFile)
	if err != nil {
		return err
	}
	defer client.Close()

	addr, err := client.ServerAddress(ctx)
	if err != nil {
		return fmt.Errorf("cannot get address: %w", err)
	}
	fmt.Printf("%s\n", addr.Address)
	return nil
}
