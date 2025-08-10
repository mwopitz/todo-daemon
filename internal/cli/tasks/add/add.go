// Package add implements the 'add' subcommand of the To-do Daemon CLI's 'tasks'
// command.
//
// The 'add' subcommend adds a new task to the to-do list, with a user-specified
// summary.
package add

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	clifmt "github.com/mwopitz/todo-daemon/internal/cli/fmt"
	"github.com/mwopitz/todo-daemon/internal/client"
	"github.com/mwopitz/todo-daemon/internal/config"
)

// Executor is used for executing the 'add' command.
type Executor struct {
	// SockFile is the path to the Unix socket file used for connecting to the
	// To-do Daemon server and creating a new task.
	SockFile string
	// TaskSummary is the summary of the to-do list task to be created.
	TaskSummary string
}

// NewExecutor creates an executor for the specified 'add' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	return &Executor{
		SockFile:    cmd.String("sock"),
		TaskSummary: cmd.StringArg("summary"),
	}, nil
}

// Execute executes the 'add' command.
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

	_, err = c.CreateTask(ctx, e.TaskSummary)
	if err != nil {
		return fmt.Errorf("cannot create task: %w", err)
	}

	tasks, err := c.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return clifmt.PrintTasks(os.Stdout, tasks)
}

// NewCommand creates a new 'add' command with the specified configuration.
func NewCommand(_ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a task to the to-do list",
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "summary"},
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
