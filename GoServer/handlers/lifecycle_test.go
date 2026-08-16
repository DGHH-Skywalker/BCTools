package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func contextWithCancel(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithCancel(r.Context())
}

func TestShutdownRejectsRemoteRequests(t *testing.T) {
	called := false
	h := NewLifecycleHandler(func() { called = true })

	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "192.168.1.50:54321"
	w := httptest.NewRecorder()

	h.HandleShutdown(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for LAN request, got %d", w.Code)
	}
	if called {
		t.Fatal("shutdown callback must NOT run for a non-localhost request")
	}
}

func TestShutdownAcceptsLocalhost(t *testing.T) {
	done := make(chan struct{})
	h := NewLifecycleHandler(func() { close(done) })

	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()

	h.HandleShutdown(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for localhost request, got %d", w.Code)
	}
	// 响应必须先于关闭动作发出，否则前端拿不到结果。
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["status"] != "closing" {
		t.Fatalf("expected status=closing, got %q", body["status"])
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown callback was never invoked")
	}
}

func TestShutdownWithoutCallbackReports501(t *testing.T) {
	h := NewLifecycleHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/shutdown", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()

	h.HandleShutdown(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when no shutdown callback is wired, got %d", w.Code)
	}
}

func TestWatchBlocksUntilClosing(t *testing.T) {
	h := NewLifecycleHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/lifecycle/watch", nil)
	w := httptest.NewRecorder()

	returned := make(chan struct{})
	go func() {
		h.HandleWatch(w, req)
		close(returned)
	}()

	// 未通知前必须一直挂起，否则前端会立刻收到假的关闭信号并关页面。
	select {
	case <-returned:
		t.Fatal("watch returned before NotifyClosing")
	case <-time.After(100 * time.Millisecond):
	}

	h.NotifyClosing()

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not return after NotifyClosing")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNotifyClosingIsIdempotent(t *testing.T) {
	h := NewLifecycleHandler(nil)
	// 托盘退出与信号退出可能并发触发；重复通知不得 panic（close of closed channel）。
	h.NotifyClosing()
	h.NotifyClosing()
	h.NotifyClosing()
}

func TestWatchReturnsWhenClientDisconnects(t *testing.T) {
	h := NewLifecycleHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/lifecycle/watch", nil)
	ctx, cancel := contextWithCancel(req)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	returned := make(chan struct{})
	go func() {
		h.HandleWatch(w, req)
		close(returned)
	}()

	cancel() // 模拟用户关掉/刷新页面

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("watch leaked: did not return after client disconnect")
	}
}
