package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"broadcast-tool/response"
	"broadcast-tool/store"
)

type SyncHandler struct {
	Store *store.Store
}

func NewSyncHandler(s *store.Store) *SyncHandler {
	return &SyncHandler{Store: s}
}

func (h *SyncHandler) HandleBackup(w http.ResponseWriter, r *http.Request) {
	settings := h.Store.GetSettingsRaw()
	if settings.AutoBackupPath == "" {
		response.WriteValidationError(w, "备份路径未设置")
		return
	}
	backupName := fmt.Sprintf("BroadcastTool_backup_%s.json", time.Now().Format("2006-01-02"))
	backupPath := filepath.Join(settings.AutoBackupPath, backupName)
	if err := h.Store.BackupTo(backupPath); err != nil {
		response.WriteValidationError(w, fmt.Sprintf("备份失败：%v", err))
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]interface{}{"success": true, "path": backupPath})
}
