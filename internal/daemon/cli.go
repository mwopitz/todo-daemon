package daemon

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/mwopitz/todo-daemon/api/todopb"
	"github.com/urfave/cli/v3"
)

// ErrAlreadyRunning is returned by [CLI.Exec] when executing the run command
// while the To-do Daemon server is already running.
var ErrAlreadyRunning = errors.New("another instance is already running")

// CLI implements the command-line interface of the To-do Daemon.
type CLI struct {
	logger  *log.Logger
	rootCmd *cli.Command
}

// NewCLI creates a new CLI instance with the specified version and an optional
// logger. If no logger is provided, the CLI will use [log.Default] instead.
func NewCLI(version string, logger *log.Logger) *CLI {
	c := &CLI{}
	c.logger = cmp.Or(logger, log.Default())
	c.rootCmd = &cli.Command{
		Name:    "todo-daemon",
		Usage:   "A daemon for managing a to-do list",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the To-do Daemon server",
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
				Name:   "status",
				Usage:  "Print the status of the To-do Daemon server",
				Action: c.printServerStatus,
			},
			{
				Name:  "tasks",
				Usage: "Manage tasks in the to-do list",
				Commands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add a new task to the to-do list",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "summary",
							},
						},
						Action: c.createTask,
					},
					{
						Name:   "list",
						Usage:  "Print all available tasks",
						Action: c.listTasks,
					},
					{
						Name:  "complete",
						Usage: "Completes the specified task",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "id",
							},
						},
						Action: c.completeTask,
					},
					{
						Name:  "delete",
						Usage: "Deletes the specified task",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name: "id",
							},
						},
						Action: c.deleteTask,
					},
				},
			},
			{
				Name:   "version",
				Usage:  "Print the version of the To-do Daemon",
				Action: c.printVersion,
			},
		},
		CommandNotFound: c.commandNotFound,
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

// Exec executes the CLI command specified by the given arguments.
func (c *CLI) Exec(ctx context.Context, args []string) error {
	return c.rootCmd.Run(ctx, args)
}

func (c *CLI) runServer(ctx context.Context, cmd *cli.Command) error {
	lockFile := cmd.String("lock")
	sockFile := cmd.String("sock")

	err := os.MkdirAll(filepath.Dir(lockFile), 0o700)
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	lock := flock.New(lockFile)
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("cannot acquire file lock: %w", err)
	}
	if !locked {
		return ErrAlreadyRunning
	}
	defer func() {
		if e := lock.Unlock(); e != nil {
			c.logger.Printf("cannot release file lock: %v", e)
		}
	}()
	c.logger.Printf("acquired file lock %s", lockFile)

	err = os.MkdirAll(filepath.Dir(sockFile), 0o700)
	if err != nil {
		return fmt.Errorf("cannot create socket directory: %w", err)
	}
	if err := os.Remove(sockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove socket file: %w", err)
	}

	// Create the To-do Daemon server and run it in a separate goroutine, so we
	// can wait until either the server stops or the context gets canceled.
	srv := NewServer(c.logger)
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve("unix", sockFile)
		close(done)
	}()

	select {
	case <-ctx.Done():
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			c.logger.Println(context.Cause(ctx))
		} else {
			c.logger.Println(err)
		}
		return srv.GracefulStop()
	case err := <-done:
		return err
	}
}

func (c *CLI) commandNotFound(_ context.Context, _ *cli.Command, name string) {
	c.logger.Printf("invalid command: '%s'", name)
}

func (c *CLI) printServerStatus(ctx context.Context, cmd *cli.Command) error {
	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if e := client.Close(); e != nil {
			c.logger.Printf("cannot gRPC close client: %v", e)
		}
	}()

	status, err := client.ServerStatus(ctx)
	if err != nil {
		return fmt.Errorf("cannot retieve server status: %w", err)
	}
	if pid := status.GetPid(); pid > 0 {
		fmt.Printf("pid=%d\n", pid)
	}
	if apiBaseURL := status.GetApiBaseUrl(); apiBaseURL != "" {
		fmt.Printf("api_base_url=%s\n", apiBaseURL)
	}
	return nil
}

func (c *CLI) createTask(ctx context.Context, cmd *cli.Command) error {
	summary := cmd.StringArg("summary")
	if summary == "" {
		return errors.New("no task summary provided")
	}

	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if e := client.Close(); e != nil {
			c.logger.Printf("cannot close gRPC client: %v", e)
		}
	}()

	_, err = client.CreateTask(ctx, summary)
	if err != nil {
		return fmt.Errorf("cannot create task: %w", err)
	}

	tasks, err := client.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return c.printTasks(tasks)
}

func (c *CLI) listTasks(ctx context.Context, cmd *cli.Command) error {
	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if e := client.Close(); e != nil {
			c.logger.Printf("cannot close gRPC client: %v", e)
		}
	}()

	tasks, err := client.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return c.printTasks(tasks)
}

func (c *CLI) completeTask(ctx context.Context, cmd *cli.Command) error {
	id := cmd.StringArg("id")
	if id == "" {
		return errors.New("no task ID provided")
	}

	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if e := client.Close(); e != nil {
			c.logger.Printf("cannot close gRPC client: %v", e)
		}
	}()

	_, err = client.CompleteTask(ctx, id)
	if err != nil {
		return fmt.Errorf("cannot complete task '%s': %w", id, err)
	}

	tasks, err := client.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return c.printTasks(tasks)
}

func (c *CLI) deleteTask(ctx context.Context, cmd *cli.Command) error {
	id := cmd.StringArg("id")
	if id == "" {
		return errors.New("no task ID provided")
	}

	sockFile := cmd.String("sock")
	client, err := NewClient("unix", sockFile, c.logger)
	if err != nil {
		return err
	}
	defer func() {
		if e := client.Close(); e != nil {
			c.logger.Printf("cannot close gRPC client: %v", e)
		}
	}()

	if err := client.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("cannot delete task: %w", err)
	}

	tasks, err := client.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("cannot retrieve tasks: %w", err)
	}

	return c.printTasks(tasks)
}

func (c *CLI) printTasks(tasks []*todopb.Task) error {
	for _, t := range tasks {
		status := ' '
		completedAt := t.GetCompletedAt().AsTime()
		if !completedAt.IsZero() && completedAt.Before(time.Now()) {
			status = '✓'
		}
		if _, err := fmt.Printf("#%s [%c] %s\n", t.GetId(), status, t.GetSummary()); err != nil {
			return err
		}
	}
	return nil
}

func (c *CLI) printVersion(_ context.Context, _ *cli.Command) error {
	_, err := fmt.Printf("go-daemon version %s\n", c.rootCmd.Version)
	return err
}
