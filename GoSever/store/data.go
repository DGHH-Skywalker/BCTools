package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"broadcast-tool/models"
)

// Store handles all data persistence with concurrent access protection
type Store struct {
	mu             sync.RWMutex
	appDataDir     string
	data           Data
	dataPath       string
	snapshotsDir   string
	deletedLogPath string
	logger         func(format string, args ...interface{})
}

// New creates a new Store, loading existing data or initializing defaults
func New(appDataDir string, logger func(format string, args ...interface{})) (*Store, error) {
	s := &Store{
		appDataDir:     appDataDir,
		dataPath:       filepath.Join(appDataDir, "data.json"),
		snapshotsDir:   filepath.Join(appDataDir, "snapshots"),
		deletedLogPath: filepath.Join(appDataDir, "deleted_songs_log.json"),
		logger:         logger,
	}
	if err := s.loadOrInit(); err != nil {
		return nil, fmt.Errorf("init store: %w", err)
	}
	return s, nil
}

// loadOrInit loads data from file or initializes with defaults on first run / corruption
func (s *Store) loadOrInit() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger("data.json not found, initializing default data")
			s.data = Data{
				DormSongs:      []models.Song{},
				BroadcastSongs: []models.Song{},
				Settings:       DefaultSettings(),
			}
			return s.atomicWriteData()
		}
		return fmt.Errorf("read data.json: %w", err)
	}

	if err := json.Unmarshal(data, &s.data); err != nil {
		backupName := fmt.Sprintf("data.json.bak.%d", time.Now().Unix())
		backupPath := filepath.Join(s.appDataDir, backupName)
		if backupErr := os.WriteFile(backupPath, data, 0644); backupErr != nil {
			s.logger("FAILED to backup corrupted data.json: %v", backupErr)
		} else {
			s.logger("corrupted data.json backed up to %s", backupName)
		}
		s.data = Data{
			DormSongs:      []models.Song{},
			BroadcastSongs: []models.Song{},
			Settings:       DefaultSettings(),
		}
		return s.atomicWriteData()
	}

	if s.data.DormSongs == nil {
		s.data.DormSongs = []models.Song{}
	}
	if s.data.BroadcastSongs == nil {
		s.data.BroadcastSongs = []models.Song{}
	}

	// Ensure computed fields are not persisted
	for i := range s.data.DormSongs {
		s.data.DormSongs[i].Weekday = ""
	}
	for i := range s.data.BroadcastSongs {
		s.data.BroadcastSongs[i].Weekday = ""
	}

	s.logger("data.json loaded: %d dorm, %d broadcast songs",
		len(s.data.DormSongs), len(s.data.BroadcastSongs))
	return nil
}

// atomicWriteData writes data atomically: tmp file + rename (Windows-safe)
func (s *Store) atomicWriteData() error {
	tmpPath := s.dataPath + ".tmp"
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, s.dataPath); err != nil {
		return fmt.Errorf("rename tmp->data.json: %w", err)
	}
	return nil
}

// --- Read operations ---

// GetDormSongs returns all dorm songs with computed weekday
func (s *Store) GetDormSongs() []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cloneSongsWithWeekday(s.data.DormSongs)
}

// GetBroadcastSongs returns all broadcast songs with computed weekday
func (s *Store) GetBroadcastSongs() []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cloneSongsWithWeekday(s.data.BroadcastSongs)
}

// GetSongByID returns a song by its ID from either list
func (s *Store) GetSongByID(id int64) (models.Song, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, song := range s.data.DormSongs {
		if song.ID == id {
			song.Weekday = calcWeekday(song.Date)
			return song, "dorm", nil
		}
	}
	for _, song := range s.data.BroadcastSongs {
		if song.ID == id {
			song.Weekday = calcWeekday(song.Date)
			return song, "broadcast", nil
		}
	}
	return models.Song{}, "", ErrNotFound
}

// GetSongsByType returns songs filtered by type and optional dates
func (s *Store) GetSongsByType(songType string, dates []string) []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var songs []models.Song
	switch songType {
	case "dorm":
		songs = s.cloneSongsWithWeekday(s.data.DormSongs)
	case "broadcast":
		songs = s.cloneSongsWithWeekday(s.data.BroadcastSongs)
	default:
		return nil
	}
	if len(dates) > 0 {
		dateSet := make(map[string]bool, len(dates))
		for _, d := range dates {
			dateSet[d] = true
		}
		filtered := make([]models.Song, 0, len(songs))
		for _, song := range songs {
			if dateSet[song.Date] {
				filtered = append(filtered, song)
			}
		}
		return filtered
	}
	return songs
}

// GetSettings returns public settings (without password hash)
func (s *Store) GetSettings() models.PublicSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return models.PublicSettings{
		TimeSlots:                 s.data.Settings.TimeSlots,
		AllowTemplateJS:           s.data.Settings.AllowTemplateJS,
		AutoBackupPath:            s.data.Settings.AutoBackupPath,
		AutoBackupEnabled:         s.data.Settings.AutoBackupEnabled,
		SilentPlaceholderDuration: s.data.Settings.SilentPlaceholderDuration,
		AdminPasswordHint:         s.data.Settings.AdminPasswordHint,
		Locale:                    s.data.Settings.Locale,
		Version:                   s.data.Settings.Version,
		DownloadURL:               s.data.Settings.DownloadURL,
	}
}

// GetAdminPasswordHash returns the stored bcrypt hash
func (s *Store) GetAdminPasswordHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings.AdminPasswordHash
}

// GetSettingsRaw returns full settings including password hash
func (s *Store) GetSettingsRaw() models.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

// GetSilentDuration returns the silent placeholder duration in seconds
func (s *Store) GetSilentDuration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data.Settings.SilentPlaceholderDuration <= 0 {
		return 30
	}
	return s.data.Settings.SilentPlaceholderDuration
}

// GetTimeSlots returns a copy of all time slots
func (s *Store) GetTimeSlots() []models.TimeSlot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slots := make([]models.TimeSlot, len(s.data.Settings.TimeSlots))
	copy(slots, s.data.Settings.TimeSlots)
	return slots
}

// IsSongsEmpty checks if both song lists are empty
func (s *Store) IsSongsEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.DormSongs) == 0 && len(s.data.BroadcastSongs) == 0
}

// --- Write operations ---

// AddSong creates a new song with auto-generated ID and timestamp
func (s *Store) AddSong(req models.CreateSongRequest) (models.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Type != "dorm" && req.Type != "broadcast" {
		return models.Song{}, ErrValidation
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return models.Song{}, ErrValidation
	}
	if strings.TrimSpace(req.Title) == "" {
		return models.Song{}, ErrValidation
	}
	sanitize := func(v string, max int) string {
		v = strings.TrimSpace(v)
		if len(v) > max {
			v = v[:max]
		}
		return v
	}
	req.Title = sanitize(req.Title, 500)
	req.Artist = sanitize(req.Artist, 500)
	req.Remark = sanitize(req.Remark, 1000)

	if req.Type == "dorm" && req.TimeSlotID != nil && strings.TrimSpace(*req.TimeSlotID) != "" {
		if s.CheckDormSlotConflict(req.Type, req.Date, *req.TimeSlotID, 0) {
			return models.Song{}, ErrSlotConflict
		}
	}

	var maxID int64
	for _, song := range s.data.DormSongs {
		if song.ID > maxID {
			maxID = song.ID
		}
	}
	for _, song := range s.data.BroadcastSongs {
		if song.ID > maxID {
			maxID = song.ID
		}
	}
	song := models.Song{
		ID:         maxID + 1,
		Date:       req.Date,
		Title:      req.Title,
		Artist:     req.Artist,
		Remark:     req.Remark,
		FilePath:   req.FilePath,
		TimeSlotID: req.TimeSlotID,
		CreatedAt:  time.Now().UTC(),
	}
	switch req.Type {
	case "dorm":
		s.data.DormSongs = append(s.data.DormSongs, song)
	case "broadcast":
		s.data.BroadcastSongs = append(s.data.BroadcastSongs, song)
	}
	if err := s.atomicWriteData(); err != nil {
		return models.Song{}, err
	}
	s.checkSnapshot()
	result := song
	result.Weekday = calcWeekday(result.Date)
	return result, nil
}

// UpdateSong updates specific fields of a song (only fields present in request)
func (s *Store) UpdateSong(id int64, req models.UpdateSongRequest) (models.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, songType := s.findSongIndex(id)
	if idx < 0 {
		return models.Song{}, ErrNotFound
	}
	var songs *[]models.Song
	switch songType {
	case "dorm":
		songs = &s.data.DormSongs
	case "broadcast":
		songs = &s.data.BroadcastSongs
	default:
		return models.Song{}, ErrNotFound
	}
	song := &(*songs)[idx]
	if req.Title != nil {
		song.Title = strings.TrimSpace(*req.Title)
	}
	if req.Artist != nil {
		song.Artist = strings.TrimSpace(*req.Artist)
	}
	if req.Remark != nil {
		song.Remark = strings.TrimSpace(*req.Remark)
	}
	if req.FilePath != nil {
		song.FilePath = *req.FilePath
	}
	if req.Date != nil {
		song.Date = *req.Date
	}
	if req.TimeSlotID != nil {
		song.TimeSlotID = *req.TimeSlotID
	}
	if songType == "dorm" && song.TimeSlotID != nil && strings.TrimSpace(*song.TimeSlotID) != "" {
		if s.CheckDormSlotConflict("dorm", song.Date, *song.TimeSlotID, id) {
			return models.Song{}, ErrSlotConflict
		}
	}
	if err := s.atomicWriteData(); err != nil {
		return models.Song{}, err
	}
	result := *song
	result.Weekday = calcWeekday(result.Date)
	return result, nil
}

// DeleteSong removes a song by ID
func (s *Store) DeleteSong(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, songType := s.findSongIndex(id)
	if idx < 0 {
		return ErrNotFound
	}
	switch songType {
	case "dorm":
		s.data.DormSongs = append(s.data.DormSongs[:idx], s.data.DormSongs[idx+1:]...)
	case "broadcast":
		s.data.BroadcastSongs = append(s.data.BroadcastSongs[:idx], s.data.BroadcastSongs[idx+1:]...)
	}
	return s.atomicWriteData()
}

// UpdateSettings updates settings fields from request (partial update)
func (s *Store) UpdateSettings(req models.UpdateSettingsRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.AllowTemplateJS != nil {
		s.data.Settings.AllowTemplateJS = *req.AllowTemplateJS
	}
	if req.AutoBackupPath != nil {
		s.data.Settings.AutoBackupPath = *req.AutoBackupPath
	}
	if req.AutoBackupEnabled != nil {
		s.data.Settings.AutoBackupEnabled = *req.AutoBackupEnabled
	}
	if req.SilentPlaceholderDuration != nil {
		v := *req.SilentPlaceholderDuration
		if v < 1 {
			v = 1
		}
		if v > 300 {
			v = 300
		}
		s.data.Settings.SilentPlaceholderDuration = v
	}
	if req.AdminPasswordHint != nil {
		s.data.Settings.AdminPasswordHint = *req.AdminPasswordHint
	}
	if req.Locale != nil {
		if *req.Locale == "zh-CN" || *req.Locale == "en" {
			s.data.Settings.Locale = *req.Locale
		}
	}
	if req.TimeSlots != nil {
		s.data.Settings.TimeSlots = *req.TimeSlots
	}
	return s.atomicWriteData()
}

// SetAdminPassword sets the bcrypt password hash
func (s *Store) SetAdminPassword(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Settings.AdminPasswordHash = hash
	return s.atomicWriteData()
}

// ReplaceSongsByType replaces all songs of a given type (for full import)
func (s *Store) ReplaceSongsByType(songType string, songs []models.Song) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if songs == nil {
		songs = []models.Song{}
	}
	switch songType {
	case "dorm":
		s.data.DormSongs = songs
	case "broadcast":
		s.data.BroadcastSongs = songs
	default:
		return ErrValidation
	}
	return s.atomicWriteData()
}

// UnassignTimeSlotIDs nullifies timeSlotId for all songs matching given IDs
func (s *Store) UnassignTimeSlotIDs(timeSlotIDs map[string]bool) ([]models.Song, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var removed []models.Song
	for i := range s.data.DormSongs {
		if s.data.DormSongs[i].TimeSlotID != nil && timeSlotIDs[*s.data.DormSongs[i].TimeSlotID] {
			removed = append(removed, s.data.DormSongs[i])
			s.data.DormSongs[i].TimeSlotID = nil
		}
	}
	for i := range s.data.BroadcastSongs {
		if s.data.BroadcastSongs[i].TimeSlotID != nil && timeSlotIDs[*s.data.BroadcastSongs[i].TimeSlotID] {
			removed = append(removed, s.data.BroadcastSongs[i])
			s.data.BroadcastSongs[i].TimeSlotID = nil
		}
	}
	if len(removed) > 0 {
		if err := s.atomicWriteData(); err != nil {
			return nil, err
		}
	}
	return removed, nil
}

// GetSongsAssignedToTimeSlotIDs returns songs assigned to given time slot IDs
func (s *Store) GetSongsAssignedToTimeSlotIDs(ids map[string]bool) []models.Song {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var affected []models.Song
	check := func(songs []models.Song) {
		for _, song := range songs {
			if song.TimeSlotID != nil && ids[*song.TimeSlotID] {
				song.Weekday = calcWeekday(song.Date)
				affected = append(affected, song)
			}
		}
	}
	check(s.data.DormSongs)
	check(s.data.BroadcastSongs)
	return affected
}

// --- Snapshot management ---

func (s *Store) checkSnapshot() {
	if err := os.MkdirAll(s.snapshotsDir, 0755); err != nil {
		s.logger("FAILED to create snapshots dir: %v", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	snapshotName := fmt.Sprintf("snapshot-%s.json", today)
	snapshotPath := filepath.Join(s.snapshotsDir, snapshotName)
	if _, err := os.Stat(snapshotPath); err == nil {
		return
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
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

// CreatePreImportSnapshot creates a snapshot before import
func (s *Store) CreatePreImportSnapshot() (string, error) {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}
	filename := fmt.Sprintf("snapshot-pre-import-%d.json", time.Now().Unix())
	path := filepath.Join(s.snapshotsDir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write pre-import snapshot: %w", err)
	}
	s.logger("pre-import snapshot: %s", filename)
	return filename, nil
}

// RestoreFromSnapshot restores data from a snapshot file
func (s *Store) RestoreFromSnapshot(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	backupPath := filepath.Join(s.appDataDir,
		fmt.Sprintf("data.json.bak.%d", time.Now().Unix()))
	currentData, _ := json.MarshalIndent(s.data, "", "  ")
	os.WriteFile(backupPath, currentData, 0644)
	snapshotPath := filepath.Join(s.snapshotsDir, filename)
	snapshotData, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	var restored Data
	if err := json.Unmarshal(snapshotData, &restored); err != nil {
		return fmt.Errorf("parse snapshot: %w", err)
	}
	s.data = restored
	if err := s.atomicWriteData(); err != nil {
		return fmt.Errorf("write restored data: %w", err)
	}
	s.logger("restored from snapshot: %s", filename)
	return nil
}

// GetSnapshots returns list of snapshot files
func (s *Store) GetSnapshots() ([]models.SnapshotInfo, error) {
	entries, err := os.ReadDir(s.snapshotsDir)
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

// cleanSnapshots removes excess snapshots (keep 30 daily, 10 pre-import)
func (s *Store) cleanSnapshots() {
	entries, err := os.ReadDir(s.snapshotsDir)
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
		os.Remove(filepath.Join(s.snapshotsDir, daily[i]))
	}
	for i := 10; i < len(preImport); i++ {
		os.Remove(filepath.Join(s.snapshotsDir, preImport[i]))
	}
}

// --- Deleted songs log ---

// AppendDeletedSongLog appends entry to deleted songs log
func (s *Store) AppendDeletedSongLog(entry models.DeletedSongLog) error {
	var entries []models.DeletedSongLog
	data, err := os.ReadFile(s.deletedLogPath)
	if err == nil {
		json.Unmarshal(data, &entries)
	}
	if entries == nil {
		entries = []models.DeletedSongLog{}
	}
	entries = append(entries, entry)
	tmpPath := s.deletedLogPath + ".tmp"
	entryData, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(tmpPath, entryData, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.deletedLogPath)
}

// ReadDeletedSongLog reads the deleted songs log
func (s *Store) ReadDeletedSongLog() ([]models.DeletedSongLog, error) {
	data, err := os.ReadFile(s.deletedLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.DeletedSongLog{}, nil
		}
		return nil, err
	}
	var entries []models.DeletedSongLog
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		return []models.DeletedSongLog{}, nil
	}
	return entries, nil
}

// --- Helpers ---

func (s *Store) findSongIndex(id int64) (int, string) {
	for i, song := range s.data.DormSongs {
		if song.ID == id {
			return i, "dorm"
		}
	}
	for i, song := range s.data.BroadcastSongs {
		if song.ID == id {
			return i, "broadcast"
		}
	}
	return -1, ""
}

func (s *Store) cloneSongsWithWeekday(songs []models.Song) []models.Song {
	result := make([]models.Song, len(songs))
	for i, song := range songs {
		song.Weekday = calcWeekday(song.Date)
		result[i] = song
	}
	return result
}

func calcWeekday(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ""
	}
	dayIndex := int(t.Weekday())
	if dayIndex == 0 {
		dayIndex = 7
	}
	weekdays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	return weekdays[dayIndex-1]
}

// GetDataPath returns the path to data.json
func (s *Store) GetDataPath() string { return s.dataPath }

// GetAppDataDir returns the app data directory
func (s *Store) GetAppDataDir() string { return s.appDataDir }

// BackupTo copies data to the specified backup path atomically
func (s *Store) BackupTo(backupPath string) error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	tmpPath := backupPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write backup tmp: %w", err)
	}
	return os.Rename(tmpPath, backupPath)
}

// GetNextID returns the next available song ID
func (s *Store) GetNextID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var maxID int64
	for _, song := range s.data.DormSongs {
		if song.ID > maxID {
			maxID = song.ID
		}
	}
	for _, song := range s.data.BroadcastSongs {
		if song.ID > maxID {
			maxID = song.ID
		}
	}
	return maxID + 1
}

// CheckDormSlotConflict checks if the given dorm date+timeSlot already has a song
func (s *Store) CheckDormSlotConflict(songType, date, timeSlotID string, excludeID int64) bool {
	if songType != "dorm" || strings.TrimSpace(timeSlotID) == "" {
		return false
	}
	for _, song := range s.data.DormSongs {
		if song.ID == excludeID {
			continue
		}
		if song.Date == date && song.TimeSlotID != nil && strings.EqualFold(strings.TrimSpace(*song.TimeSlotID), timeSlotID) {
			return true
		}
	}
	return false
}

// CheckDuplicate checks if a song with same date+title+type already exists
func (s *Store) CheckDuplicate(songType, date, title string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var songs []models.Song
	switch songType {
	case "dorm":
		songs = s.data.DormSongs
	case "broadcast":
		songs = s.data.BroadcastSongs
	default:
		return false
	}
	title = strings.TrimSpace(strings.ToLower(title))
	for _, song := range songs {
		if song.Date == date &&
			strings.EqualFold(strings.TrimSpace(song.Title), title) {
			return true
		}
	}
	return false
}
