package paths

import (
	"os"
	"path/filepath"
)

// GetAppDataDir returns %APPDATA%/BroadcastTool
func GetAppDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "BroadcastTool"), nil
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetDataFilePath returns the path to data.json
func GetDataFilePath(appDataDir string) string {
	return filepath.Join(appDataDir, "data.json")
}

// GetTempDir returns the temp directory path
func GetTempDir(appDataDir string) string {
	return filepath.Join(appDataDir, "temp")
}

// GetSnapshotsDir returns the snapshots directory path
func GetSnapshotsDir(appDataDir string) string {
	return filepath.Join(appDataDir, "snapshots")
}

// GetLogsDir returns the logs directory path
func GetLogsDir(appDataDir string) string {
	return filepath.Join(appDataDir, "logs")
}

// GetBinDir returns the bin directory path (for ffmpeg/ffprobe)
func GetBinDir(appDataDir string) string {
	return filepath.Join(appDataDir, "bin")
}

// GetDeletedSongsLogPath returns the path to deleted_songs_log.json
func GetDeletedSongsLogPath(appDataDir string) string {
	return filepath.Join(appDataDir, "deleted_songs_log.json")
}
