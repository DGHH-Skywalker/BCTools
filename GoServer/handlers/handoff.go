package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"broadcast-tool/internal/version"
	"broadcast-tool/response"
	"github.com/google/uuid"
)

// Browser clients acknowledge a prepare token only after all local drafts and
// requests have completed. Missing acknowledgements abort instead of guessing.
func (h *LifecycleHandler) HandleClient(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host && origin != "https://"+r.Host {
		http.Error(w, "cross-origin lifecycle request", http.StatusForbidden)
		return
	}
	var req struct {
		ID      string `json:"id"`
		Ack     string `json:"ack"`
		Leave   bool   `json:"leave"`
		Dirty   bool   `json:"dirty"`
		Version string `json:"version"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil || len(req.ID) < 8 || len(req.ID) > 128 {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}
	h.guard.Lock()
	defer h.guard.Unlock()
	if req.Version != "" && req.Version != version.Version {
		http.Error(w, "软件版本已变化，请刷新页面", http.StatusConflict)
		return
	}
	if _, known := h.peers[req.ID]; h.frozen && !known {
		http.Error(w, "正在切换版本，请刷新页面", http.StatusServiceUnavailable)
		return
	}
	if req.Leave {
		if !req.Dirty {
			delete(h.peers, req.ID)
			delete(h.peerSeen, req.ID)
			delete(h.peerDirty, req.ID)
		}
	} else {
		h.peers[req.ID] = req.Ack
		h.peerSeen[req.ID] = time.Now()
		h.peerDirty[req.ID] = req.Dirty
	}
	clients := []string{}
	for id, seen := range h.peerSeen {
		if time.Since(seen) < 15*time.Second {
			clients = append(clients, id)
		}
	}
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{"prepare": h.preparing, "frozen": h.frozen, "databaseId": h.dataID, "instanceId": h.instanceID, "clients": clients})
}

// MutationGate drains existing writes before freezing. It also blocks stale UI
// writes during handoff. Lifecycle acknowledgements must remain available.
func (h *LifecycleHandler) MutationGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || strings.HasPrefix(r.URL.Path, "/api/lifecycle/") || r.URL.Path == "/api/shutdown" {
			next.ServeHTTP(w, r)
			return
		}
		h.guard.Lock()
		if h.dataID != "" && r.Header.Get("X-BCTools-Instance") != h.instanceID {
			h.guard.Unlock()
			http.Error(w, "页面版本已失效，请刷新后重试", http.StatusConflict)
			return
		}
		if h.frozen {
			h.guard.Unlock()
			http.Error(w, "正在安全退出，请稍后重试", http.StatusServiceUnavailable)
			return
		}
		h.activeWrites++
		h.guard.Unlock()
		defer func() { h.guard.Lock(); h.activeWrites--; h.guard.Unlock() }()
		next.ServeHTTP(w, r)
	})
}

func (h *LifecycleHandler) PrepareShutdown(ctx context.Context) error {
	h.prepareMu.Lock()
	defer h.prepareMu.Unlock()
	h.guard.Lock()
	if h.frozen {
		h.guard.Unlock()
		return nil
	}
	h.preparing = uuid.NewString()
	token := h.preparing
	h.guard.Unlock()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		h.guard.Lock()
		ready := h.activeWrites == 0
		for _, ack := range h.peers {
			if ack != token {
				ready = false
				break
			}
		}
		if ready {
			h.frozen = true
			h.guard.Unlock()
			if h.beforeExit != nil {
				if err := h.beforeExit(ctx); err != nil {
					h.CancelShutdown()
					return fmt.Errorf("后台数据尚未保存，已取消退出：%w", err)
				}
			}
			return nil
		}
		h.guard.Unlock()
		select {
		case <-ctx.Done():
			h.guard.Lock()
			h.preparing = ""
			h.guard.Unlock()
			return fmt.Errorf("部分页面尚未确认保存（可能断网、休眠或保存失败），已取消退出，请检查打开的页面")
		case <-ticker.C:
		}
	}
}

func (h *LifecycleHandler) CancelShutdown() {
	h.guard.Lock()
	defer h.guard.Unlock()
	h.preparing = ""
	h.frozen = false
}
