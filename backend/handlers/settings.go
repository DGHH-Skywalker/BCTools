package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/response"
	"broadcast-tool/store"

	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	Store *store.Store
}

func NewSettingsHandler(s *store.Store) *SettingsHandler {
	return &SettingsHandler{Store: s}
}

func (h *SettingsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	settings := h.Store.GetSettings()
	response.WriteJSON(w, http.StatusOK, settings)
}

func (h *SettingsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "请求体格式错误")
		return
	}
	if req.TimeSlots != nil && !req.Confirmed {
		oldSlots := h.Store.GetTimeSlots()
		oldMap := make(map[string]bool)
		for _, s := range oldSlots {
			oldMap[s.ID] = true
		}
		newMap := make(map[string]bool)
		for _, s := range *req.TimeSlots {
			newMap[s.ID] = true
		}
		var deletedIDs []string
		for id := range oldMap {
			if !newMap[id] {
				deletedIDs = append(deletedIDs, id)
			}
		}
		if len(deletedIDs) > 0 {
			deletedSet := make(map[string]bool)
			for _, id := range deletedIDs {
				deletedSet[id] = true
			}
			affected := h.Store.GetSongsAssignedToTimeSlotIDs(deletedSet)
			if len(affected) > 0 {
				response.WriteJSON(w, http.StatusOK, models.SettingsConfirmResponse{
					ConfirmNeeded: true, AffectedCount: len(affected),
				})
				return
			}
		}
	}
	if req.TimeSlots != nil && req.Confirmed {
		oldSlots := h.Store.GetTimeSlots()
		oldMap := make(map[string]bool)
		for _, s := range oldSlots {
			oldMap[s.ID] = true
		}
		newSlots := *req.TimeSlots
		var deletedIDs []string
		for id := range oldMap {
			found := false
			for _, s := range newSlots {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found {
				deletedIDs = append(deletedIDs, id)
			}
		}
		if len(deletedIDs) > 0 {
			deletedSet := make(map[string]bool)
			for _, id := range deletedIDs {
				deletedSet[id] = true
			}
			removed, _ := h.Store.UnassignTimeSlotIDs(deletedSet)
			for _, song := range removed {
				label := ""
				for _, s := range oldSlots {
					if song.TimeSlotID != nil && s.ID == *song.TimeSlotID {
						label = fmt.Sprintf("%d-%s", s.DayIndex, s.Time)
						break
					}
				}
				h.Store.AppendDeletedSongLog(models.DeletedSongLog{
					OriginalID: song.ID, Date: song.Date, Title: song.Title,
					TimeSlotID: func() string {
						if song.TimeSlotID != nil {
							return *song.TimeSlotID
						}
						return ""
					}(),
					TimeSlotLabel: label, DeletedAt: time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
	}
	if req.AdminPassword != nil && *req.AdminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.AdminPassword), 12)
		if err != nil {
			response.WriteInternalError(w, "密码加密失败")
			return
		}
		if err := h.Store.SetAdminPassword(string(hash)); err != nil {
			response.WriteInternalError(w, "保存密码失败")
			return
		}
	}
	if err := h.Store.UpdateSettings(req); err != nil {
		response.WriteInternalError(w, "保存设置失败")
		return
	}
	response.WriteJSON(w, http.StatusOK, h.Store.GetSettings())
}
