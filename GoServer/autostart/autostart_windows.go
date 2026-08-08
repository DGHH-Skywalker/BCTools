//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const valueName = "BroadcastTool"

// Apply enables or disables BroadcastTool starting automatically on boot.
// It writes/updates or deletes the HKCU Run registry entry.
func Apply(enabled bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open registry run key: %w", err)
	}
	defer k.Close()

	if enabled {
		value := fmt.Sprintf(`"%s" -background`, exe)
		return k.SetStringValue(valueName, value)
	}
	return k.DeleteValue(valueName)
}

// IsEnabled checks whether the auto-start registry value is currently set.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(valueName)
	return err == nil
}
