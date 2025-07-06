package todo

import "context"

// ServerStatus holds the status of the To-do Daemon server.
type ServerStatus struct {
	// PID is the process ID of the To-do Daemon server.
	PID int
	// APIBaseURL is the base URL of the To-do Daemon's REST API.
	APIBaseURL string
}

// ServerStatusProvider is used to query the status of the To-do Daemon server.
type ServerStatusProvider interface {
	// Status returns the current status of the To-do Daemon server.
	Status(ctx context.Context) (*ServerStatus, error)
}

// ServerStatusProviderFunc is a function that implements [ServerStatusProvider].
type ServerStatusProviderFunc func(ctx context.Context) (*ServerStatus, error)

// Status returns the current status of the To-do Daemon server.
func (f ServerStatusProviderFunc) Status(ctx context.Context) (*ServerStatus, error) {
	return f(ctx)
}
