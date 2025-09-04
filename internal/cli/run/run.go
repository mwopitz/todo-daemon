// Package run implements the 'run' command of the To-do Daemon CLI.
//
// The 'run' starts the To-do Daemon server and stops the server once the
// command's context gets canceled.
package run

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/config"
	"github.com/mwopitz/todo-daemon/internal/server"
)

// ErrAlreadyRunning is returned by [Executor.Execute] when the server is
// already running.
var ErrAlreadyRunning = errors.New("another instance is already running")

// Executor is used for executing the 'run' command.
type Executor struct {
	// Lock is the file lock that the executor tries to acquire before starting
	// the server.
	Lock *flock.Flock
	// SockFile is the path to the Unix socket file that the server is supposed
	// to be listening on.
	SockFile string
}

// NewExecutor creates an executor for the specified 'run' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	return &Executor{
		Lock:     flock.New(cmd.String("lock")),
		SockFile: cmd.String("sock"),
	}, nil
}

// Execute executes the 'run' command.
func (e *Executor) Execute(ctx context.Context) error {
	unlock, err := e.lock()
	if err != nil {
		return fmt.Errorf("cannot start server: %w", err)
	}
	defer unlock()
	slog.Info("acquired file lock", "path", e.Lock.Path())

	if err := os.MkdirAll(filepath.Dir(e.SockFile), 0o700); err != nil {
		return fmt.Errorf("cannot start server: %w", err)
	}

	if err := os.Remove(e.SockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot start server: %w", err)
	}

	// Create the To-do Daemon server and run it in a separate goroutine, so we
	// can wait until either the server stops or the context gets canceled.
	srv := server.New()
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve("unix", e.SockFile)
		close(done)
	}()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			err = context.Cause(ctx)
		}
		slog.Info("stopping server...", "cause", err)
		return srv.StopGracefully()
	case err := <-done:
		return err
	}
}

func (e *Executor) lock() (func(), error) {
	err := os.MkdirAll(filepath.Dir(e.Lock.Path()), 0o700)
	if err != nil {
		return nil, err
	}
	locked, err := e.Lock.TryLock()
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrAlreadyRunning
	}
	return func() {
		if err := e.Lock.Unlock(); err != nil {
			slog.Warn("cannot release file lock", "cause", err)
		}
	}, nil
}

// NewCommand creates a new 'run' command with the specified configuration.
func NewCommand(conf *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Run the To-do Daemon server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "lock",
				Usage:     "path to the lock file",
				Value:     conf.LockFile,
				TakesFile: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			e, err := NewExecutor(cmd)
			if err != nil {
				return err
			}
			return e.Execute(ctx)
		},
	}
}
