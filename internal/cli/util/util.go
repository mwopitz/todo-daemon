// Package util provides utility functions for the To-do Daemon CLI.
package util

import (
	"fmt"
	"io"
	"time"

	"github.com/mwopitz/todo-daemon/api/todopb"
)

// PrintTasks pretty-prints the specified to-do list tasks to the given writer.
func PrintTasks(w io.Writer, tasks []*todopb.Task) error {
	for _, t := range tasks {
		status := ' '
		completedAt := t.GetCompletedAt().AsTime()
		if !completedAt.IsZero() && completedAt.Before(time.Now()) {
			status = '✓'
		}
		_, err := fmt.Fprintf(w, "#%s [%c] %s\n", t.GetId(), status, t.GetSummary())
		if err != nil {
			return err
		}
	}
	return nil
}
