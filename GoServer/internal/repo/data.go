package repo

import (
	"fmt"
	"sort"
	"time"

	"broadcast-tool/internal/version"
	"broadcast-tool/models"

	"github.com/google/uuid"
)

// Data represents the full data file structure.
type Data struct {
	DormSongs      []models.Song   `json:"dormSongs"`
	BroadcastSongs []models.Song   `json:"broadcastSongs"`
	Settings       models.Settings `json:"settings"`
}

func defaultBroadcastColumnMap() map[string]string {
	return map[string]string{
		"1": "新闻",
		"2": "文学",
		"3": "音乐",
		"4": "电影",
		"5": "新闻",
		"6": "文学",
		"7": "音乐",
	}
}

// DefaultSettings returns the default settings.
func DefaultSettings() models.Settings {
	return models.Settings{
		TimeSlots:                 DefaultTimeSlots(),
		AllowTemplateJS:           false,
		SilentPlaceholderDuration: 30,
		AdminPasswordHash:         "",
		AdminPasswordHint:         "",
		Locale:                    "zh-CN",
		Version:                   version.Version,
		DownloadURL:               "",
		BroadcastColumnMap:        defaultBroadcastColumnMap(),
		DuplicateCheckDays:        30,
	}
}

// DefaultTimeSlots creates the default time slots with UUIDs.
func DefaultTimeSlots() []models.TimeSlot {
	config := []struct {
		dayIndex int
		order    int
		time     string
	}{
		{1, 1, "06:20"}, {1, 2, "06:35"}, {1, 3, "13:55"}, {1, 4, "18:30"},
		{2, 1, "06:20"}, {2, 2, "06:35"}, {2, 3, "13:55"}, {2, 4, "18:30"},
		{3, 1, "06:20"}, {3, 2, "06:35"}, {3, 3, "13:55"}, {3, 4, "18:30"},
		{4, 1, "06:20"}, {4, 2, "06:35"}, {4, 3, "13:55"}, {4, 4, "18:30"},
		{5, 1, "06:20"}, {5, 2, "06:35"}, {5, 3, "13:55"}, {5, 4, "18:30"},
		{6, 1, "06:20"}, {6, 2, "06:35"}, {6, 3, "13:55"}, {6, 4, "18:30"},
		{7, 1, "13:55"}, {7, 2, "18:30"},
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

// EnsureDefaultTimeSlots makes sure the default time slots exist.
func EnsureDefaultTimeSlots(slots []models.TimeSlot) []models.TimeSlot {
	defaults := DefaultTimeSlots()
	existing := make(map[string]bool, len(slots))
	for _, s := range slots {
		existing[fmt.Sprintf("%d:%s", s.DayIndex, s.Time)] = true
	}
	for _, d := range defaults {
		key := fmt.Sprintf("%d:%s", d.DayIndex, d.Time)
		if !existing[key] {
			slots = append(slots, d)
		}
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].DayIndex != slots[j].DayIndex {
			return slots[i].DayIndex < slots[j].DayIndex
		}
		return slots[i].Order < slots[j].Order
	})
	return slots
}

// CalcWeekday returns the Chinese weekday for a date string (YYYY-MM-DD).
func CalcWeekday(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	dayIndex := int(t.Weekday())
	if dayIndex == 0 {
		dayIndex = 7
	}
	weekdays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	return weekdays[dayIndex-1]
}
