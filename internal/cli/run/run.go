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
	// LockFile is the path to the lock file that the executor will acquire
	// before starting the server.
	LockFile string
	// SockFile is the path to the Unix socket file that the server is supposed
	// to be listening on.
	SockFile string
}

// NewExecutor creates an executor for the specified 'run' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	return &Executor{
		LockFile: cmd.String("lock"),
		SockFile: cmd.String("sock"),
	}, nil
}

// Execute executes the 'run' command.
func (e *Executor) Execute(ctx context.Context) error {
	lockf := e.LockFile
	sockf := e.SockFile

	err := os.MkdirAll(filepath.Dir(lockf), 0o700)
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	lock := flock.New(lockf)
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	if !locked {
		return ErrAlreadyRunning
	}
	defer func() {
		if e := lock.Unlock(); e != nil {
			slog.Warn("cannot release lock file", "cause", e)
		}
	}()
	slog.Info("acquired file lock", "path", lockf)

	err = os.MkdirAll(filepath.Dir(sockf), 0o700)
	if err != nil {
		return fmt.Errorf("cannot create socket directory: %w", err)
	}
	if err := os.Remove(sockf); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove socket file: %w", err)
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
			slog.Info("stopping server...", "cause", context.Cause(ctx))
		} else {
			slog.Info("stopping server...", "cause", err)
		}
		return srv.StopGracefully()
	case err := <-done:
		return err
	}
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
