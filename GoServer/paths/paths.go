package paths

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDataRootDir 返回数据根目录：<exe 同目录>\BctoolData。
//
// 5.7.0 起所有持久化数据（data.json / MusicFiles / snapshots / logs / bin /
// decrypt-staging / temp）都集中在这个目录下，迁移时整目录拷走即可。
//
// dev（`go run`）时 exe 位于 %TEMP%\go-buildXXXX，每次构建都不同，
// 此时回退到工作目录（GoServer\BctoolData），保证开发数据可复现。
func GetDataRootDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if isGoRunTempDir(dir) {
		if wd, wdErr := os.Getwd(); wdErr == nil {
			dir = wd
		}
	}
	return filepath.Join(dir, "BctoolData"), nil
}

// isGoRunTempDir 判断目录是否为 `go run` 的临时构建目录（%TEMP%\go-buildNNN）。
func isGoRunTempDir(dir string) bool {
	tmp := os.TempDir()
	if tmp == "" {
		return false
	}
	rel, err := filepath.Rel(tmp, dir)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return !strings.HasPrefix(rel, "..") && strings.HasPrefix(rel, "go-build")
}

// GetLegacyAppDataDir 返回旧版数据目录 %APPDATA%\BroadcastTool。
// 仅用于 5.7.0 启动时的一次性自动迁移，迁移完成后原目录保留作备份。
func GetLegacyAppDataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "BroadcastTool"), nil
}

// GetLegacySongsDir 返回旧版歌曲目录（%APPDATA%\BroadcastTool\songs），
// 自动迁移时从这里读取 uuid 命名的歌曲文件。
func GetLegacySongsDir(legacyAppDataDir string) string {
	return filepath.Join(legacyAppDataDir, "songs")
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetDataFilePath returns the path to data.json
func GetDataFilePath(dataRoot string) string {
	return filepath.Join(dataRoot, "data.json")
}

// GetTempDir returns the temp directory path
func GetTempDir(dataRoot string) string {
	return filepath.Join(dataRoot, "temp")
}

// GetMusicFilesDir returns the songs directory: <root>\MusicFiles.
// 里面按「YYYY年第N周」建周文件夹，存放 SD 最终形态的歌曲文件。
func GetMusicFilesDir(dataRoot string) string {
	return filepath.Join(dataRoot, "MusicFiles")
}

// GetSongStagingDir 返回歌曲暂存目录 <root>\MusicFiles\_staging：
// 上传/解密后尚未落到周文件夹里的音频（uuid 命名）暂存在这里。
func GetSongStagingDir(dataRoot string) string {
	return filepath.Join(GetMusicFilesDir(dataRoot), "_staging")
}

// GetDecryptStagingDir returns the directory used to temporarily stage decrypted
// audio files produced by the um-react tool, so they can be handed off to the
// main app's import flow from any device on the LAN.
func GetDecryptStagingDir(dataRoot string) string {
	return filepath.Join(dataRoot, "decrypt-staging")
}

// GetSnapshotsDir returns the snapshots directory path
func GetSnapshotsDir(dataRoot string) string {
	return filepath.Join(dataRoot, "snapshots")
}

// GetLogsDir returns the logs directory path
func GetLogsDir(dataRoot string) string {
	return filepath.Join(dataRoot, "logs")
}

// GetBinDir returns the bin directory path (for ffmpeg/ffprobe)
func GetBinDir(dataRoot string) string {
	return filepath.Join(dataRoot, "bin")
}

// GetDeletedSongsLogPath returns the path to deleted_songs_log.json
func GetDeletedSongsLogPath(dataRoot string) string {
	return filepath.Join(dataRoot, "deleted_songs_log.json")
}

// GetExecutableDir returns the directory of the current executable.
func GetExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}
