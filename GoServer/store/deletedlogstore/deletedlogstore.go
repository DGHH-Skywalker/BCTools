package deletedlogstore

import (
	"encoding/json"
	"os"

	"broadcast-tool/models"
	"broadcast-tool/paths"
)

// DeletedLogStore handles the deleted songs log.
type DeletedLogStore struct {
	logPath string
}

// New creates a new DeletedLogStore for the given app data directory.
func New(appDataDir string) *DeletedLogStore {
	return &DeletedLogStore{logPath: paths.GetDeletedSongsLogPath(appDataDir)}
}

// AppendDeletedSongLog appends entry to deleted songs log.
func (s *DeletedLogStore) AppendDeletedSongLog(entry models.DeletedSongLog) error {
	var entries []models.DeletedSongLog
	data, err := os.ReadFile(s.logPath)
	if err == nil {
		json.Unmarshal(data, &entries)
	}
	if entries == nil {
		entries = []models.DeletedSongLog{}
	}
	entries = append(entries, entry)
	tmpPath := s.logPath + ".tmp"
	entryData, _ := json.MarshalIndent(entries, "", "  ")
	if err := os.WriteFile(tmpPath, entryData, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.logPath)
}

// ReadDeletedSongLog reads the deleted songs log.
func (s *DeletedLogStore) ReadDeletedSongLog() ([]models.DeletedSongLog, error) {
	data, err := os.ReadFile(s.logPath)
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
