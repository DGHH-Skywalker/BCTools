package snapshotstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"

	"github.com/google/uuid"
)

var ErrPreImportSnapshot = errors.New("导入前备份失败，已取消导入")

// SnapshotStore handles daily and pre-import snapshots.
type SnapshotStore struct {
	repo   *repo.Repository
	logger func(format string, args ...interface{})
}

// New creates a new SnapshotStore backed by the given repository.
func New(repo *repo.Repository, logger func(format string, args ...interface{})) *SnapshotStore {
	return &SnapshotStore{repo: repo, logger: logger}
}

// EnsureDailySnapshot creates a daily snapshot if one does not already exist.
func (s *SnapshotStore) EnsureDailySnapshot() {
	snapshotsDir := paths.GetSnapshotsDir(s.repo.AppDataDir())
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		s.logger("FAILED to create snapshots dir: %v", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	snapshotName := fmt.Sprintf("snapshot-%s.json", today)
	snapshotPath := filepath.Join(snapshotsDir, snapshotName)
	if _, err := os.Stat(snapshotPath); err == nil {
		return
	}
	data, err := s.repo.SnapshotData()
	if err != nil {
		s.logger("FAILED to marshal snapshot: %v", err)
		return
	}
	if err := os.WriteFile(snapshotPath, data, 0644); err != nil {
		s.logger("FAILED to write snapshot %s: %v", snapshotName, err)
		return
	}
	s.logger("snapshot created: %s", snapshotName)
	s.cleanSnapshots()
}

// CreatePreImportSnapshot backs up the state supplied by the import transaction.
func (s *SnapshotStore) CreatePreImportSnapshot(data *repo.Data) error {
	now := time.Now()
	filename := fmt.Sprintf("snapshot-pre-import-%d-%09d-%s.json", now.Unix(), now.Nanosecond(), uuid.NewString())
	if err := s.repo.SaveSnapshot(filename, data); err != nil {
		return fmt.Errorf("%w: %w", ErrPreImportSnapshot, err)
	}
	s.logger("pre-import snapshot: %s", filename)
	return nil
}

// RestoreFromSnapshot restores data from a snapshot file.
func (s *SnapshotStore) RestoreFromSnapshot(filename string) error {
	if filename != filepath.Base(filename) || !filepath.IsLocal(filename) || strings.ContainsAny(filename, `/\`) ||
		!strings.HasPrefix(filename, "snapshot-") || filepath.Ext(filename) != ".json" {
		return fmt.Errorf("invalid snapshot filename")
	}
	root, err := os.OpenRoot(paths.GetSnapshotsDir(s.repo.AppDataDir()))
	if err != nil {
		return err
	}
	defer root.Close()
	entry, err := root.Lstat(filename)
	if err != nil {
		return err
	}
	if !entry.Mode().IsRegular() {
		return fmt.Errorf("invalid snapshot file")
	}
	snapshotData, err := root.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	var restored repo.Data
	if err := json.Unmarshal(snapshotData, &restored); err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}
	return s.repo.Write(func(data *repo.Data) error {
		backupPath := filepath.Join(s.repo.AppDataDir(),
			fmt.Sprintf("data.json.bak.%d", time.Now().Unix()))
		currentData, err := json.MarshalIndent(*data, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(backupPath, currentData, 0600); err != nil {
			return err
		}
		*data = restored
		return nil
	})
}

// GetSnapshots returns list of snapshot files.
func (s *SnapshotStore) GetSnapshots() ([]models.SnapshotInfo, error) {
	entries, err := os.ReadDir(paths.GetSnapshotsDir(s.repo.AppDataDir()))
	if err != nil {
		if os.IsNotExist(err) {
			return []models.SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}
	var snapshots []models.SnapshotInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "snapshot-") {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		snapshots = append(snapshots, models.SnapshotInfo{
			Filename: entry.Name(),
			Time:     info.ModTime().Format(time.RFC3339),
			Size:     info.Size(),
		})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Filename > snapshots[j].Filename
	})
	return snapshots, nil
}

// cleanSnapshots removes excess snapshots (keep 30 daily, 10 pre-import).
func (s *SnapshotStore) cleanSnapshots() {
	entries, err := os.ReadDir(paths.GetSnapshotsDir(s.repo.AppDataDir()))
	if err != nil {
		return
	}
	var daily, preImport []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if strings.HasPrefix(entry.Name(), "snapshot-pre-import-") {
			preImport = append(preImport, entry.Name())
		} else if strings.HasPrefix(entry.Name(), "snapshot-") {
			daily = append(daily, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(daily)))
	sort.Sort(sort.Reverse(sort.StringSlice(preImport)))
	for i := 30; i < len(daily); i++ {
		os.Remove(filepath.Join(paths.GetSnapshotsDir(s.repo.AppDataDir()), daily[i]))
	}
	for i := 10; i < len(preImport); i++ {
		os.Remove(filepath.Join(paths.GetSnapshotsDir(s.repo.AppDataDir()), preImport[i]))
	}
}
