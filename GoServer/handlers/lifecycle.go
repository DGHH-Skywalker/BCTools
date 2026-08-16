package handlers

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"broadcast-tool/network"
	"broadcast-tool/response"
)

// LifecycleHandler 收拢与「进程存活 / 退出」相关的端点：健康检查、
// 关闭通知与关闭触发。原先这些散落在 main.go 里内联注册。
type LifecycleHandler struct {
	closeOnce sync.Once
	closing   chan struct{}

	// onShutdownRequest 由 main.go 注入，用于响应前端主动请求退出。
	// 为 nil 时 /api/shutdown 返回 501。
	onShutdownRequest func()
}

// NewLifecycleHandler 创建 LifecycleHandler。onShutdownRequest 可为 nil。
func NewLifecycleHandler(onShutdownRequest func()) *LifecycleHandler {
	return &LifecycleHandler{
		closing:           make(chan struct{}),
		onShutdownRequest: onShutdownRequest,
	}
}

// NotifyClosing 广播「后端即将退出」。可安全重复调用。
// 所有挂在 /api/lifecycle/watch 上的长轮询会立刻收到响应。
func (h *LifecycleHandler) NotifyClosing() {
	h.closeOnce.Do(func() { close(h.closing) })
}

// HandleHealth 返回后端存活状态。
func (h *LifecycleHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleWatch 是长轮询端点：一直挂起，直到后端准备退出才返回。
//
// 用长轮询而非定时轮询，是为了让「托盘退出 → 浏览器页自行关闭」几乎无延迟，
// 同时避免为这个极少发生的事件付出每隔几秒一次请求的代价。
func (h *LifecycleHandler) HandleWatch(w http.ResponseWriter, r *http.Request) {
	select {
	case <-h.closing:
		response.WriteJSON(w, http.StatusOK, map[string]bool{"closing": true})
	case <-r.Context().Done():
		// 客户端断开（关页面、刷新、导航），什么都不用写。
	}
}

// HandleShutdown 由前端主动请求退出后端。仅允许来自本机，
// 避免局域网内任何设备都能关掉服务。
func (h *LifecycleHandler) HandleShutdown(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !network.IsLocalhost(host) {
		log.Printf("Shutdown request rejected from %s", r.RemoteAddr)
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "仅允许本机请求关闭后端")
		return
	}
	if h.onShutdownRequest == nil {
		response.WriteError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "当前运行模式不支持关闭后端")
		return
	}
	log.Println("Shutdown requested via API")
	// 先回响应再关闭：否则连接会随服务一起断掉，前端拿不到结果。
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "closing"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.onShutdownRequest()
	}()
}

// HandleCloseBrowser 保留兼容端点：只做记录，不会退出后端。
func (h *LifecycleHandler) HandleCloseBrowser(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || !network.IsLocalhost(host) {
		log.Printf("Close-browser request rejected from %s", r.RemoteAddr)
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "仅允许本机请求")
		return
	}
	log.Println("Close-browser requested via API")
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
