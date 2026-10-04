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
	"github.com/google/uuid"
)

// Repository provides atomic, mutex-protected access to data.json.
type Repository struct {
	mu         sync.RWMutex
	appDataDir string
	dataPath   string
	data       Data
	logger     func(format string, args ...interface{})
	databaseID string
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
	idPath := filepath.Join(appDataDir, "database-id")
	b, err := os.ReadFile(idPath)
	if os.IsNotExist(err) {
		b = []byte(uuid.NewString())
		err = os.WriteFile(idPath, b, 0600)
	}
	if err != nil {
		return nil, err
	}
	r.databaseID = strings.TrimSpace(string(b))
	if _, err := uuid.Parse(r.databaseID); err != nil {
		return nil, fmt.Errorf("invalid database identity: %w", err)
	}
	return r, nil
}

func (r *Repository) DatabaseID() string { return r.databaseID }

func (r *Repository) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.atomicWrite()
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
	// Failed disk writes must not leave a phantom success in server memory.
	before, err := json.Marshal(r.data)
	if err != nil {
		return err
	}
	restore := func() { r.data = Data{}; _ = json.Unmarshal(before, &r.data) }
	if err := fn(&r.data); err != nil {
		restore()
		return err
	}
	if err := r.atomicWrite(); err != nil {
		restore()
		return err
	}
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
		return fmt.Errorf("data.json 损坏，已停止加载，原文件未覆盖：%w", err)
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
		if err := r.atomicWrite(); err != nil {
			return fmt.Errorf("migrate default time slots: %w", err)
		}
		r.logger("seeded %d default time slots for legacy data file", len(r.data.Settings.TimeSlots))
	}

	// Missing historical periods remain unknown. Creation time is only a UI
	// suggestion, never proof of the intended broadcast slot.

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
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, r.dataPath); err != nil {
		return fmt.Errorf("rename tmp->data.json: %w", err)
	}
	return nil
}
