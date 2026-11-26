//go:build !windows

package services

import (
	"os/exec"
	"runtime"
)

// openWithDefaultApp opens path in the application the desktop associates with its type. The
// path is passed as a single argument, without a shell.
func openWithDefaultApp(path string) error {
	cmd := "xdg-open"
	if runtime.GOOS == "darwin" {
		cmd = "open"
	}
	// #nosec G204 - fixed command; path is a temp file this service created
	return exec.Command(cmd, path).Start()
}
