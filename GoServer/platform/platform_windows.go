//go:build windows

package platform

import (
	"context"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func setHidden(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// Command 创建一个在 Windows 下不弹控制台窗口的 *exec.Cmd。
// 用于 cmd /c、netstat、taskkill、start、netsh 等。
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	setHidden(cmd)
	return cmd
}

// CommandContext 同 Command，但带 context。
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	setHidden(cmd)
	return cmd
}

// OpenBrowser 在系统默认浏览器中打开给定 URL。
func OpenBrowser(url string) error {
	return Command("cmd", "/c", "start", "", url).Start()
}

// WindowsInfo 返回是否运行在 Windows 上及简化的版本号。
// 只有 Windows 10 / 11 会报告为 "10" / "11"，其他版本返回空字符串。
func WindowsInfo() (isWindows bool, version string) {
	info := windows.RtlGetVersion()
	if info == nil {
		return true, ""
	}
	// Windows 11 主版本号仍是 10，靠 build >= 22000 区分。
	if info.MajorVersion == 10 {
		if info.BuildNumber >= 22000 {
			return true, "11"
		}
		return true, "10"
	}
	return true, ""
}
