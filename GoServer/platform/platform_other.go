//go:build !windows

package platform

import (
	"context"
	"os/exec"
	"runtime"
)

// Command 在非 Windows 平台是普通的 exec.Command。
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// CommandContext 在非 Windows 平台是普通的 exec.CommandContext。
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// OpenBrowser 在系统默认浏览器中打开给定 URL。
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// WindowsInfo 在非 Windows 平台恒返回 false。
func WindowsInfo() (isWindows bool, version string) {
	return false, ""
}
