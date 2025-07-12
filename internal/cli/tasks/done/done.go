// Package done implements the 'done' subcommand of the To-do Daemon CLI's
// 'tasks' command.
//
// The 'done' subcommand marks a task in the to-do list as done.
package done

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

// Executor is used for executing the 'done' command.
type Executor struct {
	// SockFile is the path to the Unix socket file used for connecting to the
	// To-do Daemon server and creating a new task.
	SockFile string
	// TaskID is the ID of the to-do list task to be completed.
	TaskID string
}

// NewExecutor creates an executor for the specified 'done' command.
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

// Execute executes the 'done' command.
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

	_, err = c.CompleteTask(ctx, e.TaskID)
	if err != nil {
		return fmt.Errorf("cannot complete task: %w", err)
	}

	tasks, err := c.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return util.PrintTasks(os.Stdout, tasks)
}

// NewCommand creates a new 'done' command with the specified configuration.
func NewCommand(_ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "done",
		Usage: "Marks a task in the to-do list as done",
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
