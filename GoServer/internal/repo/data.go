package repo

import (
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

// EnsureDefaultTimeSlots seeds the default time slots only for legacy data files
// that have no "timeSlots" field at all (nil).
//
// 重要：绝不能按「(天, 时间) 缺失就补回默认值」的方式工作。那样一来，用户在时段
// 配置里删除或修改过的默认时段（如把 06:20 改成 06:10）会在每次启动时被重新加回，
// 表现为「默认参数覆盖我的配置」。
//
//   - slots == nil     旧版数据文件没有该字段 → 补默认值
//   - len(slots) == 0  用户显式清空 → 保持为空，不复活
//   - 其他             完全尊重用户配置，只做排序
func EnsureDefaultTimeSlots(slots []models.TimeSlot) []models.TimeSlot {
	if slots == nil {
		slots = DefaultTimeSlots()
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
// weekdayNames 提为包级变量：CalcWeekday 会被每首歌调用一次，
// 原先每次调用都新建一个 slice。
var weekdayNames = [7]string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}

func CalcWeekday(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	dayIndex := int(t.Weekday())
	if dayIndex == 0 {
		dayIndex = 7
	}
	return weekdayNames[dayIndex-1]
}
