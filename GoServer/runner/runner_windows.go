//go:build windows

package runner

import (
	"context"
	"os/exec"
	"syscall"
)

func setHidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// Command creates an *exec.Cmd whose console window is hidden on Windows.
// Use this for cmd /c, netstat, taskkill, start, netsh, etc.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	setHidden(cmd)
	return cmd
}

// CommandContext is like Command but with a context.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	setHidden(cmd)
	return cmd
}
