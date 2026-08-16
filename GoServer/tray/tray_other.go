//go:build !windows

package tray

import "sync"

// 非 Windows 平台没有托盘控件。Run 仍然阻塞，这样 main.go 的生命周期
// （「主 goroutine 交给托盘，直到有人要求退出」）在各平台保持一致。
var (
	stopCh   = make(chan struct{})
	stopOnce sync.Once
)

// Run 阻塞直到 Stop() 被调用。不显示任何图标。
func Run(cfg Config) {
	<-stopCh
}

// Stop 让 Run 返回。可安全重复调用。
func Stop() {
	stopOnce.Do(func() { close(stopCh) })
}
