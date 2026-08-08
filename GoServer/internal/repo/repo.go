package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/paths"
)

// Repository provides atomic, mutex-protected access to data.json.
type Repository struct {
	mu         sync.RWMutex
	appDataDir string
	dataPath   string
	data       Data
	logger     func(format string, args ...interface{})
}

// New creates a new repository, loading existing data or initializing defaults.
func New(appDataDir string, logger func(format string, args ...interface{})) (*Repository, error) {
	r := &Repository{
		appDataDir: appDataDir,
		dataPath:   paths.GetDataFilePath(appDataDir),
		logger:     logger,
	}
	if err := r.loadOrInit(); err != nil {
		return nil, fmt.Errorf("init repo: %w", err)
	}
	return r, nil
}

// AppDataDir returns the application data directory.
func (r *Repository) AppDataDir() string { return r.appDataDir }

// DataPath returns the path to data.json.
func (r *Repository) DataPath() string { return r.dataPath }

// Read runs the given function with a read lock held.
func (r *Repository) Read(fn func(*Data)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn(&r.data)
}

// Write runs the given function with a write lock held and persists data.json atomically.
func (r *Repository) Write(fn func(*Data) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := fn(&r.data); err != nil {
		return err
	}
	return r.atomicWrite()
}

// SnapshotData returns the current data encoded as indented JSON.
func (r *Repository) SnapshotData() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return json.MarshalIndent(r.data, "", "  ")
}

func (r *Repository) loadOrInit() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			r.logger("data.json not found, initializing default data")
			r.data = Data{
				DormSongs:      []models.Song{},
				BroadcastSongs: []models.Song{},
				Settings:       DefaultSettings(),
			}
			return r.atomicWrite()
		}
		return fmt.Errorf("read data.json: %w", err)
	}

	if err := json.Unmarshal(data, &r.data); err != nil {
		backupName := fmt.Sprintf("data.json.bak.%d", time.Now().Unix())
		backupPath := filepath.Join(r.appDataDir, backupName)
		if backupErr := os.WriteFile(backupPath, data, 0644); backupErr != nil {
			r.logger("FAILED to backup corrupted data.json: %v", backupErr)
		} else {
			r.logger("corrupted data.json backed up to %s", backupName)
		}
		r.data = Data{
			DormSongs:      []models.Song{},
			BroadcastSongs: []models.Song{},
			Settings:       DefaultSettings(),
		}
		return r.atomicWrite()
	}

	if r.data.DormSongs == nil {
		r.data.DormSongs = []models.Song{}
	}
	if r.data.BroadcastSongs == nil {
		r.data.BroadcastSongs = []models.Song{}
	}

	// Ensure computed fields are not persisted.
	for i := range r.data.DormSongs {
		r.data.DormSongs[i].Weekday = ""
	}
	for i := range r.data.BroadcastSongs {
		r.data.BroadcastSongs[i].Weekday = ""
	}

	// Ensure default time slots exist even in older data files.
	originalSlotCount := len(r.data.Settings.TimeSlots)
	r.data.Settings.TimeSlots = EnsureDefaultTimeSlots(r.data.Settings.TimeSlots)
	if len(r.data.Settings.TimeSlots) != originalSlotCount {
		if err := r.atomicWrite(); err != nil {
			return fmt.Errorf("migrate default time slots: %w", err)
		}
		r.logger("migrated default time slots: added %d slots", len(r.data.Settings.TimeSlots)-originalSlotCount)
	}

	// Migrate legacy broadcast songs without a period field.
	migrated := false
	for i := range r.data.BroadcastSongs {
		if strings.TrimSpace(r.data.BroadcastSongs[i].Period) == "" {
			if r.data.BroadcastSongs[i].CreatedAt.Hour() < 12 {
				r.data.BroadcastSongs[i].Period = "noon"
			} else {
				r.data.BroadcastSongs[i].Period = "afternoon"
			}
			migrated = true
		}
	}
	if migrated {
		if err := r.atomicWrite(); err != nil {
			return fmt.Errorf("migrate broadcast periods: %w", err)
		}
		r.logger("migrated broadcast song periods")
	}

	r.logger("data.json loaded: %d dorm, %d broadcast songs",
		len(r.data.DormSongs), len(r.data.BroadcastSongs))
	return nil
}

func (r *Repository) atomicWrite() error {
	tmpPath := r.dataPath + ".tmp"
	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, r.dataPath); err != nil {
		return fmt.Errorf("rename tmp->data.json: %w", err)
	}
	return nil
}
