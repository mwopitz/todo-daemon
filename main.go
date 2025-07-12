// The program todo-daemon can either run as the To-do Daemon server or connect
// to a running To-do Daemon server instance as a client. It provides a
// command-line interface that allows users to specify whether to run the To-do
// Daemon in server mode or client mode.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mwopitz/todo-daemon/internal/cli"
	"github.com/mwopitz/todo-daemon/internal/config"
)

func main() {
	cmd := cli.NewTodoDaemonCommand(config.New())
	ctx, cancel := context.WithCancelCause(context.Background())

	errchan := make(chan error, 1)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		errchan <- cmd.Run(ctx, os.Args)
		close(errchan)
	}()

	var err error
	select {
	case err = <-errchan:
	case sig := <-sigchan:
		cancel(fmt.Errorf("received signal: %s", sig))
		err = <-errchan
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "todo-daemon: %v\n", err)
		os.Exit(1)
	}
}
