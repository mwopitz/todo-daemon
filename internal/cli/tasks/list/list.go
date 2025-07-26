// Package list implements the 'list' subcommand of the To-do Daemon CLI's
// 'tasks' command.
//
// The 'list' subcommand prints the tasks available in the to-do list to
// standard output.
package list

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/cli/util"
	"github.com/mwopitz/todo-daemon/internal/client"
	"github.com/mwopitz/todo-daemon/internal/config"
)

// Executor is used for executing the 'list' command.
type Executor struct {
	// SockFile is the path to the Unix socket file used for connecting to the
	// To-do Daemon server and creating a new task.
	SockFile string
}

// NewExecutor creates an executor for the specified 'list' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	return &Executor{
		SockFile: cmd.String("sock"),
	}, nil
}

// Execute executes the 'list' command.
func (e *Executor) Execute(ctx context.Context) error {
	c, err := client.New("unix", e.SockFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := c.Close(); err != nil {
			slog.Warn("cannot close client connection", "cause", err)
		}
	}()

	tasks, err := c.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return util.PrintTasks(os.Stdout, tasks)
}

// NewCommand creates a new 'list' command with the specified configuration.
func NewCommand(_ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "Print all tasks in the to-do list",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			e, err := NewExecutor(cmd)
			if err != nil {
				return err
			}
			return e.Execute(ctx)
		},
	}
}
