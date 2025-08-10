// Package tasks implements the 'tasks' command of the To-do Daemon CLI.
//
// The 'tasks' command provides several subcommands for managing the tasks in
// the to-do list.
package tasks

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/cli/tasks/add"
	"github.com/mwopitz/todo-daemon/internal/cli/tasks/done"
	"github.com/mwopitz/todo-daemon/internal/cli/tasks/list"
	"github.com/mwopitz/todo-daemon/internal/cli/tasks/remove"
	"github.com/mwopitz/todo-daemon/internal/config"
)

// NewCommand creates a new 'tasks' command with the specified configuration.
func NewCommand(conf *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "tasks",
		Usage: "Manage tasks in the to-do list",
		Commands: []*cli.Command{
			add.NewCommand(conf),
			list.NewCommand(conf),
			done.NewCommand(conf),
			remove.NewCommand(conf),
		},
		CommandNotFound: func(_ context.Context, _ *cli.Command, name string) {
			// revive:disable-next-line:unhandled-error
			fmt.Fprintf(os.Stderr, "todo-daemon: invalid command: '%s'\n", name)
		},
	}
}
