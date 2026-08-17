//go:build windows

package tray

// 在 init() 里给进程声明 Per-Monitor V2 DPI awareness，让所有 Win32 控件
// （systray 右键菜单、IFileOpenDialog 文件夹选择器、托盘图标等）在 1080p /
// 2K / 4K 屏上都跟系统缩放一致，不会出现「菜单文字糊、字号小」的问题。
//
// 必须在任何 HWND 出现之前调用。tray 包的 init() 在 main 之前跑，且在
// systray.Run 创建窗口前——这里就是合适的时机。
//
// 优先级（Win10 1703+ → 旧版本回退）：
//  1. SetProcessDpiAwarenessContext(DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)  ← 最佳
//  2. SetProcessDpiAwareness(PROCESS_PER_MONITOR_DPI_AWARE)                    ← Win8.1 回退
//  3. SetProcessDPIAware                                                       ← Vista+ 兜底
//
// 失败就静默：旧的 Windows 上没有这些 API，UI 仍然能用，只是显示会跟随系统
// bitmap 缩放（也就是用户之前看到的「糊」）。

import (
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 —— Win10 1703+。
	// 用 HANDLE（指针）形态传入：SetProcessDpiAwarenessContext 要的是 DPI_AWARENESS_CONTEXT
	// (void*)，传负数常量得先存到变量再取地址。
	dpiAwarenessContextPerMonitorAwareV2 = -4

	// PROCESS_PER_MONITOR_DPI_AWARE —— Win8.1 fallback
	processPerMonitorDpiAware = 2
)

var (
	moduser32 = windows.NewLazySystemDLL("user32.dll")

	procSetProcessDpiAwarenessContext = moduser32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDpiAwareness        = moduser32.NewProc("SetProcessDpiAwareness")
	procSetProcessDPIAware            = moduser32.NewProc("SetProcessDPIAware")
)

func init() {
	// 把 -4 装进一个变量才能取地址；SetProcessDpiAwarenessContext 接收
	// DPI_AWARENESS_CONTEXT（void*），Win SDK 头文件里写成 -4 字面量。
	perMonitorV2 := dpiAwarenessContextPerMonitorAwareV2

	// 1) 优先 Per-Monitor V2（Win10 1703+）
	if procSetProcessDpiAwarenessContext.Find() == nil {
		r1, _, _ := syscall.SyscallN(
			procSetProcessDpiAwarenessContext.Addr(),
			uintptr(unsafe.Pointer(&perMonitorV2)),
		)
		if r1 != 0 {
			return
		}
		log.Printf("dpi: SetProcessDpiAwarenessContext(PerMonitorV2) failed, trying fallback")
	}

	// 2) Win8.1 兜底
	if procSetProcessDpiAwareness.Find() == nil {
		hr, _, _ := syscall.SyscallN(
			procSetProcessDpiAwareness.Addr(),
			uintptr(processPerMonitorDpiAware),
		)
		if hr == 0 { // S_OK
			return
		}
		log.Printf("dpi: SetProcessDpiAwareness(PerMonitor) failed, trying Vista fallback")
	}

	// 3) Vista 兜底
	if procSetProcessDPIAware.Find() == nil {
		syscall.SyscallN(procSetProcessDPIAware.Addr())
	}
}
