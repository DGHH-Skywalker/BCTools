// Package tray provides the Windows notification-area (system tray) control.
//
// 设计约定：
//   - Run 必须在主 goroutine 调用。systray 在 init() 里 runtime.LockOSThread()，
//     其消息循环只能跑在启动它的那个 OS 线程上。
//   - Run 会阻塞直到用户点击「退出程序」或有人调用 Stop()。
//   - 非 Windows 平台是空实现（见 tray_other.go），保证跨平台 go build 可过。
package tray

// Config 描述托盘控件的行为。回调均在非主 goroutine 上执行，
// 因此实现方需自行保证并发安全。
type Config struct {
	// Tooltip 是鼠标悬停在图标上显示的文字。
	Tooltip string

	// OnOpen 在左键单击图标、或右键菜单选择「打开界面」时调用。
	OnOpen func()

	// OnExit 在右键菜单选择「退出程序」时调用，应完成后端关闭工作。
	// 它返回后 Run 才会解除阻塞。
	OnExit func()
}
