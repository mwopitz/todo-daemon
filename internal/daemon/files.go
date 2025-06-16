package daemon

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func runDir() string {
	switch runtime.GOOS {
	case "windows":
		return os.TempDir()
	default:
		return filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
}

func defaultLockFile() string {
	return filepath.Join(runDir(), "go-daemon.lock")
}

func defaultSockFile() string {
	return filepath.Join(runDir(), "go-daemon.sock")
}
