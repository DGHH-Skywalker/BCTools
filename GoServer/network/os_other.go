//go:build !windows

package network

// GetWindowsInfo is a no-op on non-Windows platforms.
func GetWindowsInfo() (isWindows bool, version string) {
	return false, ""
}
