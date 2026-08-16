package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"broadcast-tool/models"
)

func newTestRepo(t *testing.T) (*Repository, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New repo: %v", err)
	}
	return r, dir
}

func TestDefaultTimeSlots(t *testing.T) {
	r, _ := newTestRepo(t)
	var slots []models.TimeSlot
	r.Read(func(data *Data) {
		slots = data.Settings.TimeSlots
	})
	if len(slots) != 26 {
		t.Fatalf("expected 26 default time slots, got %d", len(slots))
	}
}

// 用户改过的时段必须在重启后保持原样：既不能被默认值“补回”，也不能被替换。
// 回归此前的 bug：把 06:20 改成 06:10 后重启，06:20 会被 EnsureDefaultTimeSlots 加回来。
func TestUserModifiedTimeSlotsSurviveRestart(t *testing.T) {
	r, dir := newTestRepo(t)

	custom := []models.TimeSlot{
		{ID: "u1", DayIndex: 1, Order: 1, Time: "06:10"},
		{ID: "u2", DayIndex: 1, Order: 2, Time: "06:45"},
	}
	if err := r.Write(func(data *Data) error {
		data.Settings.TimeSlots = custom
		return nil
	}); err != nil {
		t.Fatalf("write custom slots: %v", err)
	}

	// 重新打开同一数据目录，模拟重启程序
	r2, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("reopen repo: %v", err)
	}

	var got []models.TimeSlot
	r2.Read(func(data *Data) {
		got = data.Settings.TimeSlots
	})

	if len(got) != len(custom) {
		t.Fatalf("expected %d slots after restart, got %d (默认时段被重新加回了)", len(custom), len(got))
	}
	for i, want := range custom {
		if got[i].Time != want.Time || got[i].DayIndex != want.DayIndex {
			t.Fatalf("slot %d changed after restart: got %d-%s, want %d-%s",
				i, got[i].DayIndex, got[i].Time, want.DayIndex, want.Time)
		}
	}
}

// 用户显式清空所有时段后，重启不应把默认时段复活。
func TestClearedTimeSlotsStayCleared(t *testing.T) {
	r, dir := newTestRepo(t)

	if err := r.Write(func(data *Data) error {
		data.Settings.TimeSlots = []models.TimeSlot{}
		return nil
	}); err != nil {
		t.Fatalf("clear slots: %v", err)
	}

	r2, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("reopen repo: %v", err)
	}

	var got []models.TimeSlot
	r2.Read(func(data *Data) {
		got = data.Settings.TimeSlots
	})
	if len(got) != 0 {
		t.Fatalf("expected slots to stay empty, got %d (默认时段被复活了)", len(got))
	}
}

func TestWeekdayNotPersisted(t *testing.T) {
	r, dir := newTestRepo(t)
	r.Write(func(data *Data) error {
		data.DormSongs = append(data.DormSongs, models.Song{
			ID: 1, Date: "2026-07-25", Title: "Test", CreatedAt: time.Now().UTC(),
		})
		return nil
	})

	data, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	if strings.Contains(string(data), `"weekday"`) {
		t.Fatalf("weekday should not be persisted in data.json")
	}
}

func TestAtomicWrite(t *testing.T) {
	r, dir := newTestRepo(t)
	r.Write(func(data *Data) error {
		data.DormSongs = append(data.DormSongs, models.Song{
			ID: 1, Date: "2026-07-25", Title: "A", CreatedAt: time.Now().UTC(),
		})
		return nil
	})

	// Corrupt the tmp file behavior by writing invalid data directly to tmp
	tmpPath := filepath.Join(dir, "data.json.tmp")
	if err := os.WriteFile(tmpPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}

	// Another write should still succeed and leave data.json intact
	r.Write(func(data *Data) error {
		data.DormSongs = append(data.DormSongs, models.Song{
			ID: 2, Date: "2026-07-26", Title: "B", CreatedAt: time.Now().UTC(),
		})
		return nil
	})

	data, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	var payload Data
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("data.json invalid: %v", err)
	}
	if len(payload.DormSongs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(payload.DormSongs))
	}
}

func TestDataCorruptionRecovery(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(dataPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("write corrupt data: %v", err)
	}

	r, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New with corrupt data: %v", err)
	}
	var slots []models.TimeSlot
	r.Read(func(data *Data) {
		slots = data.Settings.TimeSlots
	})
	if len(slots) != 26 {
		t.Fatalf("expected default settings after corruption recovery")
	}

	// A backup should have been created
	entries, _ := os.ReadDir(dir)
	var backups int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "data.json.bak.") {
			backups++
		}
	}
	if backups != 1 {
		t.Fatalf("expected 1 backup, got %d", backups)
	}
}
