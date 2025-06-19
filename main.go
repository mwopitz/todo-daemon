// The program go-daemon can either run as the Go Daemon server or connect to a
// running Go Daemon server instance as a client. It provides a command-line
// interface that allows users to specify whether to run the Go Daemon in server
// or client mode.
package main

import (
	"context"
	"log"
	"os"

	"github.com/mwopitz/go-daemon/internal/daemon"
)

func main() {
	logger := log.New(os.Stderr, "go-daemon: ", log.Lmsgprefix)
	cli := daemon.NewCLI(logger)
	if err := cli.Run(context.Background(), os.Args); err != nil {
		logger.Fatal(err)
	}
}
