// Package fmt provides functions for formatting and printing CLI output.
package fmt

import (
	"fmt"
	"io"
	"time"

	todopb "github.com/mwopitz/todo-daemon/api/todo/v1"
)

// PrintTasks pretty-prints the specified to-do list tasks to the given writer.
func PrintTasks(w io.Writer, tasks []*todopb.Task) error {
	now := time.Now()
	for _, t := range tasks {
		status := ' '
		completedAt := t.GetCompletedAt()
		if completedAt.IsValid() && completedAt.AsTime().After(time.Unix(0, 0)) && completedAt.AsTime().Before(now) {
			status = '✓'
		}
		if _, err := fmt.Fprintf(w, "#%s [%c] %s\n", t.GetId(), status, t.GetSummary()); err != nil {
			return err
		}
	}
	return nil
}
