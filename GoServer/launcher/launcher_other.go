//go:build !windows

package launcher

import "fmt"

// KillExistingBackend is not supported on non-Windows platforms.
func KillExistingBackend(port int) error {
	return fmt.Errorf("killing existing backend is only supported on Windows")
}
