// The program go-daemon can either run as the Go Daemon server or connect to a
// running Go Daemon server instance as a client. It provides a command-line
// interface that allows users to specify whether to run the Go Daemon in server
// or client mode.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mwopitz/go-daemon/internal/daemon"
)

func main() {
	logger := log.New(os.Stderr, "go-daemon: ", log.Lmsgprefix)
	cli := daemon.NewCLI(logger)
	ctx, cancel := context.WithCancelCause(context.Background())

	errchan := make(chan error, 1)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		errchan <- cli.Run(ctx, os.Args)
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
		logger.Fatal(err)
	}
}
