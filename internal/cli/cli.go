// Package cli implements the command-line interface of the To-do Daemon.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/cli/run"
	"github.com/mwopitz/todo-daemon/internal/cli/status"
	"github.com/mwopitz/todo-daemon/internal/cli/tasks"
	"github.com/mwopitz/todo-daemon/internal/config"
	"github.com/mwopitz/todo-daemon/internal/version"
)

// NewTodoDaemonCommand creates the root command of the To-do Daemon CLI with
// the specified configuration.
func NewTodoDaemonCommand(conf *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "todo-daemon",
		Version: version.Semantic(),
		Usage:   "A daemon for managing a to-do list",
		Commands: []*cli.Command{
			run.NewCommand(conf),
			status.NewCommand(conf),
			tasks.NewCommand(conf),
		},
		CommandNotFound: func(_ context.Context, _ *cli.Command, name string) {
			fmt.Fprintf(os.Stderr, "todo-daemon: invalid command: '%s'\n", name)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "sock",
				Usage:     "path to the socket file",
				Value:     conf.SockFile,
				TakesFile: true,
			},
		},
	}
}
