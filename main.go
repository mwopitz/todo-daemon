package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofrs/flock"
	"github.com/mwopitz/go-daemon/client"
	"github.com/mwopitz/go-daemon/server"
	"github.com/urfave/cli/v3"
)

func main() {
	log.SetFlags(0)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("fatal: cannot determine home directory: %v", err)
	}

	lockFile := filepath.Join(home, ".go-daemon.lock")
	sockFile := filepath.Join(home, ".go-daemon.sock")

	cmd := &cli.Command{
		Name: "go-daemon",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the go-daemon server",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return run(ctx, lockFile, sockFile)
				},
			},
			{
				Name:  "address",
				Usage: "Get the address of the go-daemon server",
				Action: func(ctx context.Context, _ *cli.Command) error {
					return address(ctx, sockFile)
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, lockFile, sockFile string) error {
	lock := flock.New(lockFile)
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("cannot acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("cannot acquire lock file %s; already running?", lockFile)
	}
	defer lock.Unlock()

	if err := os.Remove(sockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot remove socket file: %w", err)
	}

	srv := server.New()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errc := make(chan error, 1)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() {
		errc <- srv.Serve("unix", sockFile)
		close(errc)
	}()

	select {
	case <-ctx.Done():
		log.Printf("go-daemon: %v", ctx.Err())
		return srv.GracefulStop()
	case sig := <-sigc:
		log.Printf("go-daemon: %s", sig)
		return srv.GracefulStop()
	case err := <-errc:
		return err
	}
}

func address(ctx context.Context, socket string) error {
	c, err := client.New("unix", socket)
	if err != nil {
		return err
	}
	defer c.Close()

	addr, err := c.ServerAddress(ctx)
	if err != nil {
		return fmt.Errorf("cannot get address: %w", err)
	}
	fmt.Printf("%s\n", addr.Address)
	return nil
}
