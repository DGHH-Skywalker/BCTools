package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandoffNeedsEveryClientAckAndBlocksWrites(t *testing.T) {
	h := NewLifecycleHandler(nil)
	h.peers["tab-one"] = ""
	h.peers["tab-two"] = ""
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { done <- h.PrepareShutdown(ctx) }()
	var token string
	for token == "" {
		h.guard.RLock()
		token = h.preparing
		h.guard.RUnlock()
		time.Sleep(time.Millisecond)
	}
	h.guard.Lock()
	h.peers["tab-one"] = token
	h.guard.Unlock()
	select {
	case err := <-done:
		t.Fatalf("second client was not saved: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	h.guard.Lock()
	h.peers["tab-two"] = token
	h.guard.Unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.MutationGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("mutation accepted after handoff") })).ServeHTTP(w, httptest.NewRequest("PUT", "/api/songs/1", nil))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func TestHandoffWaitsForInflightWriteAndCanCancel(t *testing.T) {
	h := NewLifecycleHandler(nil)
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		h.MutationGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release })).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("PUT", "/api/settings", nil))
		close(finished)
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := h.PrepareShutdown(ctx); err == nil {
		t.Fatal("acknowledged an in-flight write")
	}
	close(release)
	<-finished
	if err := h.PrepareShutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFailedPersistenceCancelsShutdownAndUnfreezes(t *testing.T) {
	h := NewLifecycleHandler(func() { t.Error("must not exit after disk failure") })
	h.SetPersistence("database", func(context.Context) error { return fmt.Errorf("disk full") })
	r := httptest.NewRequest("POST", "/api/shutdown", nil)
	r.RemoteAddr = "127.0.0.1:1"
	w := httptest.NewRecorder()
	h.HandleShutdown(w, r)
	if w.Code != 409 || h.frozen || h.preparing != "" {
		t.Fatalf("failed save did not cancel safely: %d %v", w.Code, h.frozen)
	}
}

func TestNewClientCannotRegisterAfterFreeze(t *testing.T) {
	h := NewLifecycleHandler(nil)
	if err := h.PrepareShutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.HandleClient(w, httptest.NewRequest("POST", "/api/lifecycle/client", strings.NewReader(`{"id":"new-client","dirty":true}`)))
	if w.Code != 503 {
		t.Fatalf("late client accepted: %d", w.Code)
	}
}

func TestHandoffTimeoutDoesNotFreezeOrExit(t *testing.T) {
	h := NewLifecycleHandler(func() { t.Error("must not exit") })
	h.peers["offline-tab"] = ""
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if h.PrepareShutdown(ctx) == nil {
		t.Fatal("expected unacknowledged client error")
	}
	if h.frozen || h.preparing != "" {
		t.Fatal("aborted switch locked UI")
	}
}

func TestCrossOriginCannotUnregisterClient(t *testing.T) {
	h := NewLifecycleHandler(nil)
	h.peers["client-1234"] = ""
	r := httptest.NewRequest("POST", "http://localhost/api/lifecycle/client", strings.NewReader(`{"id":"client-1234","leave":true}`))
	r.Header.Set("Origin", "http://evil.invalid")
	w := httptest.NewRecorder()
	h.HandleClient(w, r)
	if w.Code != 403 || len(h.peers) != 1 {
		t.Fatal("cross-origin client removal")
	}
}
