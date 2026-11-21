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
	if logger == nil {
		logger = func(string, ...interface{}) {}
	}
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
	return r.writeLocked(fn, nil, nil)
}

func (r *Repository) WriteWithMoves(fn func(*Data) error, moves []FileMove) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.writeLocked(fn, moves, nil)
}

func (r *Repository) writeLocked(fn func(*Data) error, moves []FileMove, extra map[string]json.RawMessage) error {
	data, err := json.Marshal(r.data)
	if err != nil {
		return err
	}
	var next Data
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	if err := fn(&next); err != nil {
		return err
	}
	if next.NextSongID < r.data.NextSongID {
		next.NextSongID = r.data.NextSongID
	}
	includeSongID(&next, next.DormSongs)
	includeSongID(&next, next.BroadcastSongs)
	if err := r.persist(next, moves, extra); err != nil {
		return err
	}
	r.data = next
	return nil
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

	if err := os.MkdirAll(r.appDataDir, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(r.appDataDir)
	if err != nil {
		return err
	}
	defer root.Close()
	var pending fileTransaction
	if err := readJSON(root, journalName, &pending); err == nil {
		if err := replay(root, pending); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

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

	var header storageHeader
	headerErr := json.Unmarshal(data, &header)
	if headerErr == nil && header.SchemaVersion != 0 {
		r.data, err = loadStorage(root, header)
		if err != nil {
			return err
		}
	} else if err := json.Unmarshal(data, &r.data); err != nil {
		if _, archiveErr := root.Lstat(archiveName); archiveErr == nil {
			return fmt.Errorf("corrupt storage header: %w", err)
		}
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

	// Seed default time slots only for legacy data files missing the field entirely.
	// 用 nil 判断而非数量比较：用户清空或改动时段后不应被默认值覆盖。
	needSeedTimeSlots := r.data.Settings.TimeSlots == nil
	r.data.Settings.TimeSlots = EnsureDefaultTimeSlots(r.data.Settings.TimeSlots)
	if needSeedTimeSlots {
		r.logger("seeded %d default time slots for legacy data file", len(r.data.Settings.TimeSlots))
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
		r.logger("migrated broadcast song periods")
	}

	r.logger("data.json loaded: %d dorm, %d broadcast songs",
		len(r.data.DormSongs), len(r.data.BroadcastSongs))
	if header.SchemaVersion == 0 {
		if err := atomicFile(root, "data.v1-backup.json", data); err != nil {
			return err
		}
	}
	includeSongID(&r.data, r.data.DormSongs)
	includeSongID(&r.data, r.data.BroadcastSongs)
	if header.SchemaVersion == 0 || needSeedTimeSlots || migrated {
		return r.atomicWrite()
	}
	return nil
}

func (r *Repository) atomicWrite() error {
	return r.persist(r.data, nil, nil)
}
