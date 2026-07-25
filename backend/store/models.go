package store

import (
	"broadcast-tool/models"
	"github.com/google/uuid"
)

// Data represents the full data file structure
type Data struct {
	DormSongs      []models.Song   `json:"dormSongs"`
	BroadcastSongs []models.Song   `json:"broadcastSongs"`
	Settings       models.Settings `json:"settings"`
}

// DefaultSettings returns the default settings
func DefaultSettings() models.Settings {
	return models.Settings{
		TimeSlots:                 DefaultTimeSlots(),
		AllowTemplateJS:           false,
		AutoBackupPath:            "",
		AutoBackupEnabled:         false,
		SilentPlaceholderDuration: 30,
		AdminPasswordHash:         "",
		AdminPasswordHint:         "",
		Locale:                    "zh-CN",
		Version:                   "1.0.0",
		DownloadURL:               "",
	}
}

// DefaultTimeSlots creates the default time slots with UUIDs
func DefaultTimeSlots() []models.TimeSlot {
	config := []struct {
		dayIndex int
		order    int
		time     string
	}{
		{1, 1, "06:20"}, {1, 2, "06:35"}, {1, 3, "13:50"}, {1, 4, "18:30"},
		{2, 1, "06:20"}, {2, 2, "06:35"}, {2, 3, "13:50"}, {2, 4, "18:30"},
		{3, 1, "06:20"}, {3, 2, "06:35"}, {3, 3, "13:50"}, {3, 4, "18:30"},
		{4, 1, "06:20"}, {4, 2, "06:35"}, {4, 3, "13:50"}, {4, 4, "18:30"},
		{5, 1, "06:20"}, {5, 2, "06:35"}, {5, 3, "13:50"}, {5, 4, "18:30"},
		{6, 1, "06:20"}, {6, 2, "06:35"}, {6, 3, "13:50"}, {6, 4, "18:30"},
		{7, 1, "18:30"},
	}

	slots := make([]models.TimeSlot, len(config))
	for i, c := range config {
		slots[i] = models.TimeSlot{
			ID:       uuid.New().String(),
			DayIndex: c.dayIndex,
			Order:    c.order,
			Time:     c.time,
		}
	}
	return slots
}
