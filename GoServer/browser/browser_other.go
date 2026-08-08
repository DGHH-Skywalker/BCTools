//go:build !windows

package browser

import (
	"os/exec"
	"runtime"
)

// Open opens the given URL in the system default browser.
func Open(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
