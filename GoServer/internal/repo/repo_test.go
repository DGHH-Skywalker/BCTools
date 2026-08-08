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
