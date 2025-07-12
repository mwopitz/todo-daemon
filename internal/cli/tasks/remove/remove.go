// Package remove implements the 'remove' subcommand of the To-do Daemon CLI's
// 'tasks' command.
//
// The 'remove' subcommand removes a task from the to-do list.
package remove

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/cli/util"
	"github.com/mwopitz/todo-daemon/internal/client"
	"github.com/mwopitz/todo-daemon/internal/config"
)

// Executor is used for executing the 'remove' command.
type Executor struct {
	// SockFile is the path to the Unix socket file used for connecting to the
	// To-do Daemon server and creating a new task.
	SockFile string
	// TaskID is the ID of the to-do list task to be removed.
	TaskID string
}

// NewExecutor creates an executor for the specified 'remove' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	taskID := cmd.StringArg("id")
	if taskID == "" {
		return nil, errors.New("no task ID specified")
	}
	return &Executor{
		SockFile: cmd.String("sock"),
		TaskID:   taskID,
	}, nil
}

// Execute executes the 'remove' command.
func (e *Executor) Execute(ctx context.Context) error {
	c, err := client.New("unix", e.SockFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeerr := c.Close(); closeerr != nil {
			slog.Warn("cannot close client connection", "cause", closeerr)
		}
	}()

	err = c.DeleteTask(ctx, e.TaskID)
	if err != nil {
		return fmt.Errorf("cannot delete task: %w", err)
	}

	tasks, err := c.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return util.PrintTasks(os.Stdout, tasks)
}

// NewCommand creates a new 'remove' command with the specified configuration.
func NewCommand(_ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "Removes a task from the to-do list",
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "id"},
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
