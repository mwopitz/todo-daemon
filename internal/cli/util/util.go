// Package util provides utility functions for the To-do Daemon CLI.
package util

import (
	"fmt"
	"io"
	"log"
	"time"

	todopb "github.com/mwopitz/todo-daemon/internal/api/todo/v1"
)

// PrintTasks pretty-prints the specified to-do list tasks to the given writer.
func PrintTasks(w io.Writer, tasks []*todopb.Task) error {
	now := time.Now()
	for _, t := range tasks {
		status := ' '
		completedAt := t.GetCompletedAt()
		if completedAt.IsValid() && completedAt.AsTime().Before(now) {
			log.Printf("%s is before %s", completedAt, now)
			status = '✓'
		}
		if _, err := fmt.Fprintf(w, "#%s [%c] %s\n", t.GetId(), status, t.GetSummary()); err != nil {
			return err
		}
	}
	return nil
}
