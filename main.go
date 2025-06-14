package main

import (
	"context"
	"log"
	"os"

	"github.com/mwopitz/go-daemon/cli"
)

func main() {
	logger := log.New(os.Stderr, "go-daemon: ", 0)
	cli := cli.New(logger)
	if err := cli.Run(context.Background(), os.Args); err != nil {
		logger.Fatal(err)
	}
}
