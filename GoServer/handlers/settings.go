package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/services"
	"broadcast-tool/store/deletedlogstore"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/store/songstore"
)

type SettingsHandler struct {
	Settings        *settingstore.SettingsStore
	Songs           *songstore.SongStore
	DeletedLog      *deletedlogstore.DeletedLogStore
	settingsService *services.SettingsUpdateService
}

func NewSettingsHandler(settings *settingstore.SettingsStore, songs *songstore.SongStore, deletedLog *deletedlogstore.DeletedLogStore) *SettingsHandler {
	return &SettingsHandler{
		Settings:        settings,
		Songs:           songs,
		DeletedLog:      deletedLog,
		settingsService: services.NewSettingsUpdateService(settings, songs, deletedLog),
	}
}

func (h *SettingsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	settings := h.Settings.GetSettings()
	response.WriteJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}

	if affected := h.settingsService.CheckTimeSlotDeletion(req); affected > 0 {
		response.WriteJSON(w, http.StatusOK, models.SettingsConfirmResponse{
			ConfirmNeeded: true,
			AffectedCount: affected,
		})
		return
	}

	if err := h.settingsService.ApplyTimeSlotDeletion(req); err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}

	if req.AdminPassword != nil {
		if err := h.settingsService.UpdateAdminPassword(*req.AdminPassword); err != nil {
			response.WriteInternalError(w, err.Error())
			return
		}
	}

	if err := h.Settings.UpdateSettings(req); err != nil {
		response.WriteInternalError(w, "保存设置失败")
		return
	}

	if err := h.settingsService.ApplyAutoStart(req.AutoStartEnabled); err != nil {
		log.Printf("WARNING: failed to apply auto-start setting: %v", err)
	}

	response.WriteJSON(w, http.StatusOK, h.Settings.GetSettings())
}
