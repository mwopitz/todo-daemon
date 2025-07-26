// Package status implements the 'status' command of the To-do Daemon CLI.
//
// The 'status' command queries the status of the To-do Daemon server and prints
// the status to standard output.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/mwopitz/todo-daemon/internal/client"
	"github.com/mwopitz/todo-daemon/internal/config"
)

const (
	outputFormatJSON = "json"
)

// Executor is used for executing the 'status' command.
type Executor struct {
	// SockFile is the path to the Unix socket file used for connecting to the
	// To-do Daemon server.
	SockFile string
	// OutputFormat specifies the format for printing the status to standard
	// output.
	OutputFormat string
}

// NewExecutor creates an executor for the specified 'status' command.
func NewExecutor(cmd *cli.Command) (*Executor, error) {
	return &Executor{
		SockFile:     cmd.String("sock"),
		OutputFormat: cmd.String("format"),
	}, nil
}

// Execute executes the 'status' command.
func (o *Executor) Execute(ctx context.Context) error {
	c, err := client.New("unix", o.SockFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := c.Close(); err != nil {
			slog.Warn("cannot close client connection", "cause", err)
		}
	}()

	status, err := c.ServerStatus(ctx)
	if err != nil {
		return err
	}

	switch format := o.OutputFormat; format {
	case outputFormatJSON:
		err = json.NewEncoder(os.Stdout).Encode(status)
		if err != nil {
			return fmt.Errorf("cannot print status: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("invalid output format: %s", format)
	}
}

// NewCommand creates a new 'status' command with the specified configuration.
func NewCommand(_ *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Print the status of the To-do Daemon server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "format",
				Usage:     "the output format",
				Value:     outputFormatJSON,
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
