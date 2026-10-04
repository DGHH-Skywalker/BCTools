package services

import (
	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"broadcast-tool/paths"
)

// LegacyCandidates only inspects known application locations; never scans disks.
func LegacyCandidates(executable string) []string {
	candidates := []string{filepath.Join(filepath.Dir(executable), "BctoolData")}
	if local, err := os.UserCacheDir(); err == nil {
		candidates = append(candidates, filepath.Join(local, "Programs", "BCTools", "BctoolData"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		candidates = append(candidates, filepath.Join(pf, "BCTools", "BctoolData"))
	}
	if old, err := paths.GetLegacyAppDataDir(); err == nil {
		candidates = append(candidates, old)
	}
	return candidates
}

// PrepareFixedData copies, never moves. Multiple distinct databases require an
// explicit choice; timestamps/record counts cannot establish which is authoritative.
// data.json is committed LAST, making an interrupted copy safely retryable.
func PrepareFixedData(root string, candidates []string, explicit string) error {
	if _, err := os.Stat(paths.GetDataFilePath(root)); err == nil {
		if explicit != "" {
			return fmt.Errorf("固定数据目录已有数据；请使用备份恢复流程，不允许启动时覆盖：%s", root)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if explicit != "" {
		candidates = []string{explicit}
	}
	seen := map[[32]byte]string{}
	for _, dir := range candidates {
		b, err := os.ReadFile(paths.GetDataFilePath(dir))
		if os.IsNotExist(err) && explicit == "" {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := decodeDataSource(b); err != nil {
			return fmt.Errorf("数据格式异常，未迁移 %s：%w", dir, err)
		}
		key := sha256.Sum256(b)
		if _, exists := seen[key]; !exists {
			seen[key] = dir
		}
	}
	if len(seen) > 1 {
		sources := []DataSource{}
		for _, dir := range seen {
			sources = append(sources, InspectDataSource(dir))
		}
		sort.Slice(sources, func(i, j int) bool { return sources[i].Directory < sources[j].Directory })
		return &DataConflictError{Sources: sources}
	}
	for _, source := range seen {
		rel, err := filepath.Rel(source, root)
		if err != nil || rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return fmt.Errorf("迁移目标不能位于来源内部：%s", source)
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			return err
		}
		// Copy all historical resources including snapshots and recovery files.
		err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("拒绝迁移符号链接：%s", path)
			}
			if rel == "data.json" {
				return nil
			}
			target := filepath.Join(root, rel)
			if entry.IsDir() {
				return os.MkdirAll(target, 0700)
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("非常规文件：%s", path)
			}
			return copyFileAtomic(path, target)
		})
		if err != nil {
			return err
		}
		// Pre-5.7 bare audio paths resolve in MusicFiles/_staging in the new version.
		oldSongs := filepath.Join(root, "songs")
		if entries, err := os.ReadDir(oldSongs); err == nil {
			staging := paths.GetSongStagingDir(root)
			if err := os.MkdirAll(staging, 0700); err != nil {
				return err
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if err := copyFileAtomic(filepath.Join(oldSongs, entry.Name()), filepath.Join(staging, entry.Name())); err != nil {
					return err
				}
			}
		}
		if err := copyFileAtomic(paths.GetDataFilePath(source), filepath.Join(root, "data.pre-fixed-migration.json")); err != nil {
			return err
		}
		return copyFileAtomic(paths.GetDataFilePath(source), paths.GetDataFilePath(root))
	}
	return nil
}

type DataConflictError struct{ Sources []DataSource }

func (e *DataConflictError) Error() string {
	return "发现多份不同歌单，请选择首次迁移来源；原文件未修改"
}

type DataSource struct {
	ID             string `json:"id"`
	Directory      string `json:"directory"`
	Digest         string `json:"digest"`
	DormCount      int    `json:"dormCount"`
	BroadcastCount int    `json:"broadcastCount"`
	FirstDate      string `json:"firstDate"`
	LastDate       string `json:"lastDate"`
	MissingFiles   int    `json:"missingFiles"`
	Error          string `json:"error,omitempty"`
}

func (s DataSource) Description() string {
	return fmt.Sprintf("%s\n宿舍 %d 首，播音 %d 首\n日期 %s 至 %s；缺少音频 %d 项", s.Directory, s.DormCount, s.BroadcastCount, s.FirstDate, s.LastDate, s.MissingFiles)
}

func decodeDataSource(b []byte) (repo.Data, error) {
	var fields map[string]json.RawMessage
	var data repo.Data
	if err := json.Unmarshal(b, &fields); err != nil {
		return data, err
	}
	if fields["broadcastSongs"] == nil || fields["dormSongs"] == nil {
		return data, fmt.Errorf("缺少歌单字段")
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return data, err
	}
	for _, songs := range [][]models.Song{data.DormSongs, data.BroadcastSongs} {
		for _, s := range songs {
			if _, err := time.Parse("2006-01-02", s.Date); err != nil {
				return data, fmt.Errorf("歌曲 %d 日期无效", s.ID)
			}
			if s.FilePath != "" && !safeRelativePath(s.FilePath) {
				return data, fmt.Errorf("歌曲 %d 音频路径超出数据目录", s.ID)
			}
		}
	}
	return data, nil
}

func safeRelativePath(path string) bool {
	path = strings.ReplaceAll(path, "\\", "/")
	clean := filepath.Clean(filepath.FromSlash(path))
	return !filepath.IsAbs(clean) && filepath.VolumeName(clean) == "" && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)) && !strings.Contains(path, ":")
}

func InspectDataSource(dir string) DataSource {
	abs, _ := filepath.Abs(dir)
	s := DataSource{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ToLower(abs)))), Directory: abs}
	b, err := os.ReadFile(paths.GetDataFilePath(abs))
	if err != nil {
		s.Error = err.Error()
		return s
	}
	data, err := decodeDataSource(b)
	if err != nil {
		s.Error = err.Error()
		return s
	}
	s.Digest = fmt.Sprintf("%x", sha256.Sum256(b))
	s.DormCount = len(data.DormSongs)
	s.BroadcastCount = len(data.BroadcastSongs)
	for _, songs := range [][]models.Song{data.DormSongs, data.BroadcastSongs} {
		for _, song := range songs {
			if s.FirstDate == "" || song.Date < s.FirstDate {
				s.FirstDate = song.Date
			}
			if song.Date > s.LastDate {
				s.LastDate = song.Date
			}
			if song.FilePath != "" {
				if _, err := sourceAudioPath(abs, song.FilePath); err != nil {
					s.MissingFiles++
				}
			}
		}
	}
	return s
}

func sourceAudioPath(root, path string) (string, error) {
	if !safeRelativePath(path) {
		return "", fmt.Errorf("invalid audio path")
	}
	path = filepath.FromSlash(strings.ReplaceAll(path, "\\", "/"))
	for _, candidate := range []string{filepath.Join(root, path), filepath.Join(paths.GetMusicFilesDir(root), path), filepath.Join(root, "songs", path), filepath.Join(paths.GetSongStagingDir(root), path)} {
		if info, err := os.Lstat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("音频文件缺失：%s", path)
}

func RecordDataSources(root string, candidates []string) error {
	file := filepath.Join(root, "legacy-sources.json")
	var old []string
	if b, err := os.ReadFile(file); err == nil {
		if err = json.Unmarshal(b, &old); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	dirs := []string{}
	for _, dir := range append(old, candidates...) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		if strings.EqualFold(abs, root) || seen[strings.ToLower(abs)] {
			continue
		}
		if _, err := os.Stat(paths.GetDataFilePath(abs)); err != nil {
			continue
		}
		seen[strings.ToLower(abs)] = true
		dirs = append(dirs, abs)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(dirs, "", "  ")
	if err != nil {
		return err
	}
	return atomicBytes(file, b)
}

func ListDataSources(root string) ([]DataSource, error) {
	var dirs []string
	b, err := os.ReadFile(filepath.Join(root, "legacy-sources.json"))
	if os.IsNotExist(err) {
		return []DataSource{}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &dirs); err != nil {
		return nil, err
	}
	result := []DataSource{}
	for _, dir := range dirs {
		result = append(result, InspectDataSource(dir))
	}
	return result, nil
}

func atomicBytes(path string, b []byte) error {
	f, err := os.OpenFile(path+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
