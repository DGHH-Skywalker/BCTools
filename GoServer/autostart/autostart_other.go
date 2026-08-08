//go:build !windows

package autostart

// Apply is a no-op on non-Windows platforms.
func Apply(enabled bool) error { return nil }

// IsEnabled is a no-op on non-Windows platforms.
func IsEnabled() bool { return false }
