//go:build windows

package tray

import (
	"embed"
	"log"
	"sync"

	"fyne.io/systray"
)

//go:embed icon-sys.ico
var iconFS embed.FS

// stopOnce 保证 Stop 幂等：托盘菜单退出与信号退出可能同时发生。
var stopOnce sync.Once

// Run 显示托盘图标并阻塞运行消息循环，直到用户选择退出或调用 Stop()。
// 必须在主 goroutine 调用。
func Run(cfg Config) {
	systray.Run(func() { onReady(cfg) }, func() {})
}

// Stop 主动收起托盘并让 Run 返回。可安全重复调用。
func Stop() {
	stopOnce.Do(systray.Quit)
}

func onReady(cfg Config) {
	if icon, err := iconFS.ReadFile("icon-sys.ico"); err != nil {
		// 图标读不出来不该拖垮整个程序：菜单仍然可用，只是没有图形。
		log.Printf("tray: failed to read embedded icon: %v", err)
	} else {
		systray.SetIcon(icon)
	}
	systray.SetTooltip(cfg.Tooltip)

	// 不注册 SetOnTapped——左键只显示图标，没有回调。
	// 配合 tray/dpi_windows.go 的 Per-Monitor V2，右键菜单在 2K/4K 屏上
	// 自动跟随系统缩放，不再糊。
	mOpen := systray.AddMenuItem("打开界面", "在浏览器中打开小播点歌工具")
	systray.AddSeparator()
	mExit := systray.AddMenuItem("退出程序", "关闭后端服务并退出")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				invoke("open", cfg.OnOpen)
			case <-mExit.ClickedCh:
				// 先跑完关闭逻辑，再收起托盘，避免进程在 HTTP 优雅关闭
				// 完成前就被消息循环退出带走。
				invoke("exit", cfg.OnExit)
				Stop()
				return
			}
		}
	}()
}

// invoke 执行回调并拦截 panic，防止一次点击把整个消息循环打死。
func invoke(name string, fn func()) {
	if fn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("tray: %s handler panicked: %v", name, r)
		}
	}()
	fn()
}
