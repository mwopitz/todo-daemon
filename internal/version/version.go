// Package version provides the version of the To-do Daemon.
package version

import "fmt"

var (
	// Major is the major version of the To-do Daemon.
	Major = 0
	// Minor is the minor version of the To-do Daemon.
	Minor = 0
	// Patch is the patch versein of the To-do Daemon.
	Patch = 0
)

// Semantic returns the semantic version of the To-do Daemon.
func Semantic() string {
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}
