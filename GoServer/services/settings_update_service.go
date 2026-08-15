package services

import (
	"fmt"
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
