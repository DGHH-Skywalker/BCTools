package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"broadcast-tool/models"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	return s, dir
}

func TestDefaultTimeSlots(t *testing.T) {
	s, _ := newTestStore(t)
	settings := s.GetSettings()
	if len(settings.TimeSlots) != 25 {
		t.Fatalf("expected 25 default time slots, got %d", len(settings.TimeSlots))
	}
}

func TestWeekdayNotPersisted(t *testing.T) {
	s, dir := newTestStore(t)
	_, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Test"})
	if err != nil {
		t.Fatalf("AddSong: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	if strings.Contains(string(data), `"weekday"`) {
		t.Fatalf("weekday should not be persisted in data.json")
	}
}

func TestSongCRUD(t *testing.T) {
	s, _ := newTestStore(t)
	song, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Test", Artist: "Artist"})
	if err != nil {
		t.Fatalf("AddSong: %v", err)
	}
	if song.ID != 1 {
		t.Fatalf("expected ID 1, got %d", song.ID)
	}
	if song.Weekday == "" {
		t.Fatalf("expected computed weekday in response")
	}

	songs := s.GetSongsByType("dorm", nil)
	if len(songs) != 1 {
		t.Fatalf("expected 1 dorm song, got %d", len(songs))
	}

	newTitle := "Updated"
	updated, err := s.UpdateSong(song.ID, models.UpdateSongRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateSong: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected updated title")
	}

	if err := s.DeleteSong(song.ID); err != nil {
		t.Fatalf("DeleteSong: %v", err)
	}
	if len(s.GetSongsByType("dorm", nil)) != 0 {
		t.Fatalf("expected 0 songs after delete")
	}
}

func TestAtomicWrite(t *testing.T) {
	s, dir := newTestStore(t)
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A"}); err != nil {
		t.Fatalf("AddSong: %v", err)
	}

	// Corrupt the tmp file behavior by writing invalid data directly to tmp
	tmpPath := filepath.Join(dir, "data.json.tmp")
	if err := os.WriteFile(tmpPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}

	// Another write should still succeed and leave data.json intact
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-26", Title: "B"}); err != nil {
		t.Fatalf("AddSong after tmp corruption: %v", err)
	}

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

func TestSnapshotLifecycle(t *testing.T) {
	s, dir := newTestStore(t)
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A"}); err != nil {
		t.Fatalf("AddSong: %v", err)
	}

	snaps, err := s.GetSnapshots()
	if err != nil {
		t.Fatalf("GetSnapshots: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}

	filename := snaps[0].Filename
	if _, err := os.Stat(filepath.Join(dir, "snapshots", filename)); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}

	if err := s.RestoreFromSnapshot(filename); err != nil {
		t.Fatalf("RestoreFromSnapshot: %v", err)
	}
}

func TestDataCorruptionRecovery(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(dataPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("write corrupt data: %v", err)
	}

	s, err := New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New with corrupt data: %v", err)
	}
	if len(s.GetSettings().TimeSlots) != 25 {
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
