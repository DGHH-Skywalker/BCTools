package settingstore

import (
	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"os"
	"path/filepath"
	"testing"
)

func TestSlotDeletionAndSettingsRollbackTogether(t *testing.T) {
	root := t.TempDir()
	r, err := repo.New(root, func(string, ...interface{}) {})
	if err != nil {
		t.Fatal(err)
	}
	oldID := "original"
	oldSlots := []models.TimeSlot{{ID: oldID, DayIndex: 1, Order: 1, Time: "06:20"}}
	r.Write(func(d *repo.Data) error {
		d.Settings.TimeSlots = oldSlots
		d.DormSongs = []models.Song{{ID: 1, Date: "2026-10-05", TimeSlotID: &oldID}}
		return nil
	})
	s := New(r)
	empty := []models.TimeSlot{}
	if _, err := s.UpdateSettingsAtomically(models.UpdateSettingsRequest{TimeSlots: &empty}, nil); err == nil {
		t.Fatal("deleted without confirmation")
	}
	// Force the atomic write to fail after the transaction changes its data.
	os.Mkdir(filepath.Join(root, "data.json.tmp"), 0700)
	if _, err := s.UpdateSettingsAtomically(models.UpdateSettingsRequest{TimeSlots: &empty, Confirmed: true}, nil); err == nil {
		t.Fatal("expected disk failure")
	}
	r.Read(func(d *repo.Data) {
		if len(d.Settings.TimeSlots) != 1 || d.DormSongs[0].TimeSlotID == nil {
			t.Fatal("partial update survived failed write")
		}
	})
	os.Remove(filepath.Join(root, "data.json.tmp"))
	removed, err := s.UpdateSettingsAtomically(models.UpdateSettingsRequest{TimeSlots: &empty, Confirmed: true}, nil)
	if err != nil || len(removed) != 1 {
		t.Fatalf("commit failed: %v", err)
	}
	reopened, err := repo.New(root, func(string, ...interface{}) {})
	if err != nil {
		t.Fatal(err)
	}
	reopened.Read(func(d *repo.Data) {
		if len(d.Settings.TimeSlots) != 0 || d.DormSongs[0].TimeSlotID != nil {
			t.Fatal("transaction not durable")
		}
	})
}
