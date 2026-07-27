package handlers

import (
	"net/http"

	"broadcast-tool/response"
	"broadcast-tool/store"
)

type SnapshotHandler struct {
	Store *store.Store
}

func NewSnapshotHandler(s *store.Store) *SnapshotHandler {
	return &SnapshotHandler{Store: s}
}

func (h *SnapshotHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.Store.GetSnapshots()
	if err != nil {
		response.WriteInternalError(w, "获取快照列表失败")
		return
	}
	response.WriteJSON(w, http.StatusOK, snapshots)
}

func (h *SnapshotHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		response.WriteValidationError(w, "filename 参数不能为空")
		return
	}
	if err := h.Store.RestoreFromSnapshot(filename); err != nil {
		response.WriteInternalError(w, "恢复快照失败")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}
