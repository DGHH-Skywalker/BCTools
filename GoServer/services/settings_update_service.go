package services

import (
	"fmt"
	"regexp"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/store/deletedlogstore"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/store/songstore"

	"golang.org/x/crypto/bcrypt"
)

// SettingsUpdateService encapsulates the business logic for updating settings,
// including time-slot deletion side effects and password changes.
type SettingsUpdateService struct {
	Settings   *settingstore.SettingsStore
	Songs      *songstore.SongStore
	DeletedLog *deletedlogstore.DeletedLogStore
}

// NewSettingsUpdateService creates a new SettingsUpdateService.
func NewSettingsUpdateService(
	settings *settingstore.SettingsStore,
	songs *songstore.SongStore,
	deletedLog *deletedlogstore.DeletedLogStore,
) *SettingsUpdateService {
	return &SettingsUpdateService{Settings: settings, Songs: songs, DeletedLog: deletedLog}
}

// CheckTimeSlotDeletion returns the number of songs affected by deleting time slots.
// If no slots are being deleted, or no songs are affected, it returns 0.
func (s *SettingsUpdateService) CheckTimeSlotDeletion(req models.UpdateSettingsRequest) int {
	if req.TimeSlots == nil || req.Confirmed {
		return 0
	}
	oldSlots := s.Settings.GetTimeSlots()
	deletedSet := deletedSlotIDs(oldSlots, *req.TimeSlots)
	if len(deletedSet) == 0 {
		return 0
	}
	affected := s.Songs.GetSongsAssignedToTimeSlotIDs(deletedSet)
	return len(affected)
}

// ReconcileTimeSlots preserves an existing slot identity whenever a submitted
// slot still represents the same day and position. This protects assignments
// when a UI recreates a row while editing a time or copying a day configuration.
func (s *SettingsUpdateService) ReconcileTimeSlots(req *models.UpdateSettingsRequest) {
	if req.TimeSlots == nil {
		return
	}
	*req.TimeSlots = reconcileSlots(s.Settings.GetTimeSlots(), *req.TimeSlots)
}

func reconcileSlots(oldSlots, submitted []models.TimeSlot) []models.TimeSlot {
	result := append([]models.TimeSlot{}, submitted...)
	known := map[string]models.TimeSlot{}
	used := map[string]bool{}
	for _, slot := range oldSlots {
		known[slot.ID] = slot
	}
	// Reserve ALL explicit identities before trying to match recreated rows.
	for _, slot := range result {
		if old, ok := known[slot.ID]; ok && old.DayIndex == slot.DayIndex {
			used[slot.ID] = true
		}
	}
	for i := range result {
		slot := &result[i]
		if _, ok := known[slot.ID]; ok {
			continue
		}
		var candidates []string
		for _, old := range oldSlots {
			if !used[old.ID] && old.DayIndex == slot.DayIndex && old.Time == slot.Time {
				candidates = append(candidates, old.ID)
			}
		}
		if len(candidates) == 1 {
			slot.ID = candidates[0]
			used[slot.ID] = true
		}
		// Never infer identity from order: inserting/deleting/reordering changes it.
		// A real edit keeps its original ID; an unmatched new slot is genuinely new.
	}
	return result
}

func ValidateTimeSlots(old, submitted []models.TimeSlot) error {
	known := map[string]models.TimeSlot{}
	for _, slot := range old {
		known[slot.ID] = slot
	}
	ids := map[string]bool{}
	positions := map[string]bool{}
	clock := regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)
	for _, slot := range submitted {
		if slot.ID == "" || ids[slot.ID] {
			return fmt.Errorf("时段 ID 为空或重复")
		}
		ids[slot.ID] = true
		if slot.DayIndex < 1 || slot.DayIndex > 7 || slot.Order < 1 || !clock.MatchString(slot.Time) {
			return fmt.Errorf("时段星期、顺序或时间无效")
		}
		key := fmt.Sprintf("%d:%d", slot.DayIndex, slot.Order)
		if positions[key] {
			return fmt.Errorf("同一天的时段顺序重复")
		}
		positions[key] = true
		if old, ok := known[slot.ID]; ok && old.DayIndex != slot.DayIndex {
			return fmt.Errorf("已有时段不能跨星期复用 ID")
		}
	}
	return nil
}

// ApplyTimeSlotDeletion unassigns songs from deleted time slots and logs them.
func (s *SettingsUpdateService) ApplyTimeSlotDeletion(req models.UpdateSettingsRequest) error {
	if req.TimeSlots == nil || !req.Confirmed {
		return nil
	}
	oldSlots := s.Settings.GetTimeSlots()
	newSlots := *req.TimeSlots
	deletedSet := deletedSlotIDs(oldSlots, newSlots)
	if len(deletedSet) == 0 {
		return nil
	}
	removed, err := s.Songs.UnassignTimeSlotIDs(deletedSet)
	if err != nil {
		return err
	}
	for _, song := range removed {
		label := ""
		for _, slot := range oldSlots {
			if song.TimeSlotID != nil && slot.ID == *song.TimeSlotID {
				label = fmt.Sprintf("%d-%s", slot.DayIndex, slot.Time)
				break
			}
		}
		timeSlotID := ""
		if song.TimeSlotID != nil {
			timeSlotID = *song.TimeSlotID
		}
		s.DeletedLog.AppendDeletedSongLog(models.DeletedSongLog{
			OriginalID:    song.ID,
			Date:          song.Date,
			Title:         song.Title,
			TimeSlotID:    timeSlotID,
			TimeSlotLabel: label,
			DeletedAt:     time.Now().UTC().Format(time.RFC3339),
		})
	}
	return nil
}

// UpdateAdminPassword hashes and stores a new admin password if provided.
func (s *SettingsUpdateService) UpdateAdminPassword(password string) error {
	if password == "" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("密码加密失败")
	}
	if err := s.Settings.SetAdminPassword(string(hash)); err != nil {
		return fmt.Errorf("保存密码失败")
	}
	return nil
}

func deletedSlotIDs(oldSlots, newSlots []models.TimeSlot) map[string]bool {
	oldMap := make(map[string]bool)
	for _, s := range oldSlots {
		oldMap[s.ID] = true
	}
	newMap := make(map[string]bool)
	for _, s := range newSlots {
		newMap[s.ID] = true
	}
	deleted := make(map[string]bool)
	for id := range oldMap {
		if !newMap[id] {
			deleted[id] = true
		}
	}
	return deleted
}

func slotPosition(slot models.TimeSlot) string {
	return fmt.Sprintf("%d:%d", slot.DayIndex, slot.Order)
}
