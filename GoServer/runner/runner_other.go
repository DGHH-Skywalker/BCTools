//go:build !windows

package runner

import (
	"context"
	"os/exec"
)

// Command is a no-op wrapper on non-Windows platforms.
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// CommandContext is a no-op wrapper on non-Windows platforms.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
