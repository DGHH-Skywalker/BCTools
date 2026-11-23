package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"broadcast-tool/internal/contracts"
	"broadcast-tool/internal/kugoudb"
	"broadcast-tool/response"
)

func (h *DecryptHandler) HandleKugouStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	custom, count := h.Kugou.Status()
	response.WriteJSON(w, http.StatusOK, contracts.KugouDatabaseStatus{
		CanSelect: kugoudb.CanSelect, Custom: custom, KeyCount: count,
	})
}

func (h *DecryptHandler) HandleKugouSelect(w http.ResponseWriter, r *http.Request) {
	if !kugoudb.CanSelect {
		response.WriteError(w, http.StatusNotImplemented, "KUGOU_PICKER_UNAVAILABLE", "本机数据库选择支持 Windows")
		return
	}
	if !h.kugouPicker.TryLock() {
		response.WriteError(w, http.StatusConflict, "KUGOU_PICKER_BUSY", "已有酷狗数据库文件选择窗口")
		return
	}
	defer h.kugouPicker.Unlock()
	path, err := selectKugouDatabase()
	if err != nil {
		log.Printf("选择酷狗数据库: %v", err)
		response.WriteInternalError(w, "无法打开酷狗数据库选择窗口")
		return
	}
	if path == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := h.Kugou.Select(r.Context(), path); err != nil {
		log.Printf("导入酷狗数据库: %v", err)
		response.WriteError(w, http.StatusUnprocessableEntity, "INVALID_KUGOU_DATABASE", "无法读取所选酷狗数据库，请确认文件格式并关闭酷狗客户端后重试")
		return
	}
	h.HandleKugouStatus(w, r)
}

func (h *DecryptHandler) HandleKugouReset(w http.ResponseWriter, r *http.Request) {
	if err := h.Kugou.Select(r.Context(), ""); err != nil {
		log.Printf("恢复酷狗数据库自动查找: %v", err)
		response.WriteInternalError(w, "无法更新酷狗数据库来源")
		return
	}
	h.HandleKugouStatus(w, r)
}

func (h *DecryptHandler) HandleKugouKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var query contracts.KugouKeyQuery
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&query); err != nil ||
		strings.TrimSpace(query.AudioHash) == "" || len(query.AudioHash) > 256 {
		response.WriteValidationError(w, "无效的音频标识")
		return
	}
	key := h.Kugou.Key(query.AudioHash)
	if key == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	response.WriteJSON(w, http.StatusOK, contracts.KugouKeyResult{Key: key})
}
