package snapshotstore

import (
	"os"
	"path/filepath"
	"testing"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
)

func newTestSnapshotStore(t *testing.T) (*SnapshotStore, *repo.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := repo.New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New repo: %v", err)
	}
	return New(r, func(format string, args ...interface{}) { t.Logf(format, args...) }), r, dir
}

func TestSnapshotLifecycle(t *testing.T) {
	s, _, dir := newTestSnapshotStore(t)
	s.EnsureDailySnapshot()

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

func TestPreImportSnapshot(t *testing.T) {
	s, r, _ := newTestSnapshotStore(t)
	r.Write(func(data *repo.Data) error {
		data.DormSongs = append(data.DormSongs, models.Song{ID: 1, Date: "2026-07-25", Title: "A"})
		return nil
	})
	filename, err := s.CreatePreImportSnapshot()
	if err != nil {
		t.Fatalf("CreatePreImportSnapshot: %v", err)
	}
	if filename == "" {
		t.Fatalf("expected filename")
	}
}
