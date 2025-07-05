package todo

import "context"

// ServerStatus holds the status of the To-do Daemon server.
type ServerStatus struct {
	// PID is the process ID of the To-do Daemon server.
	PID int
	// APIBaseURL is the base URL of the To-do Daemon's REST API.
	APIBaseURL string
}

type ServerStatusProvider interface {
	Status(ctx context.Context) (*ServerStatus, error)
}

type ServerStatusProviderFunc func(ctx context.Context) (*ServerStatus, error)

func (f ServerStatusProviderFunc) Status(ctx context.Context) (*ServerStatus, error) {
	return f(ctx)
}
