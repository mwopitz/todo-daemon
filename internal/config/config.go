// Package config provides the configuration for the To-do Daemon.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Config holds the configuration of the To-do Daemon.
type Config struct {
	// LockFile holds the path to the lock file used by the To-do Daemon server
	// to ensure that only a single instance of the server can be running.
	LockFile string `json:"lock_file"`
	// SockFile holds the path to the UNIX socket file used for communication
	// between the To-do Daemon server process and the command processes.
	SockFile string `json:"sock_file"`
}

// New returns a configuration with default values.
func New() *Config {
	return &Config{
		LockFile: defaultLockFile(),
		SockFile: defaultSockFile(),
	}
}

func runDir() string {
	switch runtime.GOOS {
	case "windows":
		return os.TempDir()
	default:
		return filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
}

func defaultLockFile() string {
	return filepath.Join(runDir(), "todo-daemon.lock")
}

func defaultSockFile() string {
	return filepath.Join(runDir(), "todo-daemon.sock")
}
