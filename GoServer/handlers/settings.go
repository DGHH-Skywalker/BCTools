package handlers

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"

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
	Library         *services.MusicLibrary
	settingsService *services.SettingsUpdateService
}

func NewSettingsHandler(settings *settingstore.SettingsStore, songs *songstore.SongStore, deletedLog *deletedlogstore.DeletedLogStore, library *services.MusicLibrary) *SettingsHandler {
	return &SettingsHandler{
		Settings:        settings,
		Songs:           songs,
		DeletedLog:      deletedLog,
		Library:         library,
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
	h.settingsService.ReconcileTimeSlots(&req)
	oldSlots := h.Settings.GetTimeSlots()
	if req.TimeSlots != nil {
		if err := services.ValidateTimeSlots(oldSlots, *req.TimeSlots); err != nil {
			response.WriteValidationError(w, err.Error())
			return
		}
	}

	if affected := h.settingsService.CheckTimeSlotDeletion(req); affected > 0 {
		response.WriteJSON(w, http.StatusOK, models.SettingsConfirmResponse{
			ConfirmNeeded: true,
			AffectedCount: affected,
		})
		return
	}

	var passwordHash *string
	if req.AdminPassword != nil && *req.AdminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.AdminPassword), 12)
		if err != nil {
			response.WriteInternalError(w, "密码加密失败")
			return
		}
		value := string(hash)
		passwordHash = &value
	}
	removed, err := h.Settings.UpdateSettingsAtomically(req, passwordHash)
	if err != nil {
		response.WriteInternalError(w, "保存设置失败")
		return
	}
	for _, song := range removed {
		id := ""
		label := ""
		if song.TimeSlotID != nil {
			id = *song.TimeSlotID
		}
		for _, slot := range oldSlots {
			if slot.ID == id {
				label = fmt.Sprintf("%d-%s", slot.DayIndex, slot.Time)
				break
			}
		}
		h.DeletedLog.AppendDeletedSongLog(models.DeletedSongLog{OriginalID: song.ID, Date: song.Date, Title: song.Title, TimeSlotID: id, TimeSlotLabel: label, DeletedAt: time.Now().UTC().Format(time.RFC3339)})
	}

	// 时段配置（新增/删除/排序）会改变周文件夹的编号锚点，全量重排。
	if req.TimeSlots != nil && h.Library != nil {
		if err := h.Library.SyncAll(); err != nil {
			log.Printf("musiclibrary: resync after settings update failed: %v", err)
		}
	}

	response.WriteJSON(w, http.StatusOK, h.Settings.GetSettings())
}
