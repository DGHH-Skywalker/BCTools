package repo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"broadcast-tool/models"

	"github.com/google/uuid"
)

type Recovery struct {
	ID         string                                `json:"id"`
	CreatedAt  time.Time                             `json:"createdAt"`
	SongCount  int                                   `json:"songCount"`
	FileCount  int                                   `json:"fileCount"`
	Restored   bool                                  `json:"restored,omitempty"`
	RestoredAt *time.Time                            `json:"restoredAt,omitempty"`
	Songs      songLists                             `json:"songs"`
	Weeks      map[string]map[string]models.WeekFile `json:"weeks"`
	Files      []FileMove                            `json:"files"`
}

var numberedAudio = regexp.MustCompile(`^[0-9]{2,}\.mp3$`)
var stagedAudio = regexp.MustCompile(`^(?:song-[0-9]+|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\.mp3$`)

func retentionDir(id string) string {
	return filepath.Join("recycle", id)
}

func (r *Repository) Retention() models.RetentionSettings {
	var result models.RetentionSettings
	r.Read(func(d *Data) { result = d.Settings.Retention.Effective() })
	return result
}

func (r *Repository) SetRetention(settings models.RetentionSettings) error {
	if settings.AudioWeeks < 1 || settings.AudioWeeks > 520 || settings.MetadataYears < 1 || settings.MetadataYears > 50 {
		return errors.New("音频保留期为 1–520 周，歌曲信息保留期为 1–50 年")
	}
	return r.Write(func(d *Data) error {
		d.Settings.Retention = settings
		return nil
	})
}

func (r *Repository) Collect(now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	root, err := os.OpenRoot(r.appDataDir)
	if err != nil {
		return err
	}
	defer root.Close()
	settings := r.data.Settings.Retention.Effective()
	start, _ := weekStart(weekForDate(now.Format(time.DateOnly)))
	audioCutoff := start.AddDate(0, 0, -(settings.AudioWeeks-1)*7)
	metaCutoff := now.AddDate(-settings.MetadataYears, 0, 0).Format(time.DateOnly)
	today := now.Format(time.DateOnly)
	recovery := Recovery{ID: uuid.NewString(), CreatedAt: now, Weeks: make(map[string]map[string]models.WeekFile)}
	keepFile := make(map[string]bool)
	recycleFile := make(map[string]bool)
	filter := func(songs []models.Song, archived *[]models.Song) []models.Song {
		result := make([]models.Song, 0, len(songs))
		for _, song := range songs {
			name, managed := managedAudio(song.FilePath)
			week, valid := weekStart(weekForDate(song.Date))
			held := song.RetentionHoldUntil > today
			remove := valid && !held && song.Date < metaCutoff
			clearAudio := valid && !held && managed && (remove || week.Before(audioCutoff))
			if remove || clearAudio {
				*archived = append(*archived, song)
				if managed {
					recycleFile[name] = true
				}
			}
			if !remove {
				if clearAudio {
					song.FilePath = ""
				} else if managed {
					keepFile[name] = true
				}
				result = append(result, song)
			}
		}
		return result
	}
	dorm := filter(r.data.DormSongs, &recovery.Songs.DormSongs)
	broadcast := filter(r.data.BroadcastSongs, &recovery.Songs.BroadcastSongs)
	entries, err := recoveries(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.RestoredAt != nil && now.Before(entry.RestoredAt.AddDate(0, 0, 7)) {
			for _, move := range entry.Files {
				keepFile[move.From] = true
			}
		}
	}
	for name, index := range r.data.Weeks {
		week, valid := weekStart(name)
		if !valid || !week.Before(audioCutoff) {
			continue
		}
		for number, file := range index {
			path, safe := managedAudio(name + "/" + number + ".mp3")
			if !safe || keepFile[path] {
				continue
			}
			if recovery.Weeks[name] == nil {
				recovery.Weeks[name] = make(map[string]models.WeekFile)
			}
			recovery.Weeks[name][number] = file
			recycleFile[path] = true
		}
	}
	if err := collectOrphanAudio(root, audioCutoff, recycleFile); err != nil {
		return err
	}
	for name := range recycleFile {
		if keepFile[name] {
			continue
		}
		info, err := root.Lstat(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("无法归档非普通音频文件 %s", name)
		}
		recovery.Files = append(recovery.Files, FileMove{
			From: name, To: filepath.Join(retentionDir(recovery.ID), "files", name),
		})
	}
	recovery.SongCount = len(recovery.Songs.DormSongs) + len(recovery.Songs.BroadcastSongs)
	recovery.FileCount = len(recovery.Files)
	if recovery.SongCount > 0 || recovery.FileCount > 0 || len(recovery.Weeks) > 0 {
		b, err := json.Marshal(recovery)
		if err != nil {
			return err
		}
		err = r.writeLocked(func(d *Data) error {
			d.DormSongs, d.BroadcastSongs = dorm, broadcast
			for name, files := range recovery.Weeks {
				for id := range files {
					delete(d.Weeks[name], id)
				}
				if len(d.Weeks[name]) == 0 {
					delete(d.Weeks, name)
				}
			}
			return nil
		}, recovery.Files, map[string]json.RawMessage{filepath.Join(retentionDir(recovery.ID), "record.json"): b})
		if err != nil {
			return err
		}
	}
	if info, err := root.Lstat("data.v1-backup.json"); err == nil && info.Mode().IsRegular() && !now.Before(info.ModTime().AddDate(0, 0, 7)) {
		if err := root.Remove("data.v1-backup.json"); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		purgeAfter := entry.CreatedAt.AddDate(0, 0, 7)
		if entry.RestoredAt != nil {
			purgeAfter = entry.RestoredAt.AddDate(0, 0, 7)
		}
		if !now.Before(purgeAfter) {
			if err := root.RemoveAll(retentionDir(entry.ID)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Old recordings with generated filenames can be left behind by interrupted
// week exports or abandoned staging. Include them in the recoverable cleanup.
func collectOrphanAudio(root *os.Root, cutoff time.Time, recycle map[string]bool) error {
	directories, err := fs.ReadDir(root.FS(), "MusicFiles")
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range directories {
		if !entry.IsDir() {
			continue
		}
		staged := entry.Name() == "_staging"
		week, valid := weekStart(entry.Name())
		if !staged && (!valid || !week.Before(cutoff)) {
			continue
		}
		dir := "MusicFiles/" + entry.Name()
		files, err := fs.ReadDir(root.FS(), dir)
		if err != nil {
			return err
		}
		for _, file := range files {
			if !file.Type().IsRegular() {
				continue
			}
			if staged {
				info, err := file.Info()
				if err != nil || !stagedAudio.MatchString(file.Name()) || !info.ModTime().Before(cutoff) {
					continue
				}
			} else if !numberedAudio.MatchString(file.Name()) {
				continue
			}
			recycle[filepath.Join(dir, file.Name())] = true
		}
	}
	return nil
}

func recoveries(root *os.Root) ([]Recovery, error) {
	dir, err := root.Open("recycle")
	if errors.Is(err, fs.ErrNotExist) {
		return []Recovery{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	result := make([]Recovery, 0)
	for _, entry := range entries {
		id, err := uuid.Parse(entry.Name())
		if err != nil || id.String() != entry.Name() || !entry.IsDir() {
			continue
		}
		var record Recovery
		if err := readJSON(root, filepath.Join(retentionDir(entry.Name()), "record.json"), &record); err != nil {
			return nil, err
		}
		if record.ID != entry.Name() || record.CreatedAt.IsZero() {
			return nil, fmt.Errorf("invalid recovery %s", entry.Name())
		}
		result = append(result, record)
	}
	return result, nil
}

func (r *Repository) Recoveries() ([]Recovery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	root, err := os.OpenRoot(r.appDataDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return recoveries(root)
}

func (r *Repository) RestoreRecovery(id string, now time.Time) error {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id {
		return errors.New("无效的回收记录")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	root, err := os.OpenRoot(r.appDataDir)
	if err != nil {
		return err
	}
	defer root.Close()
	recordName := filepath.Join(retentionDir(id), "record.json")
	var record Recovery
	if err := readJSON(root, recordName, &record); err != nil {
		return err
	}
	if record.ID != id || record.Restored || !now.Before(record.CreatedAt.AddDate(0, 0, 7)) {
		return errors.New("回收记录已失效")
	}
	moves := make([]FileMove, 0, len(record.Files))
	for _, move := range record.Files {
		if !filepath.IsLocal(move.From) || move.To != filepath.Join(retentionDir(id), "files", move.From) {
			return errors.New("回收记录的文件路径无效")
		}
		if _, err := root.Lstat(move.From); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("恢复目标已经存在或无法访问: %s", move.From)
		}
		moves = append(moves, FileMove{From: move.To, To: move.From})
	}
	holdUntil := now.AddDate(0, 0, 7).Format(time.DateOnly)
	restoreSongs := func(target *[]models.Song, songs []models.Song) {
		for _, saved := range songs {
			saved.RetentionHoldUntil = holdUntil
			found := false
			for i := range *target {
				if (*target)[i].ID == saved.ID {
					found = true
					if (*target)[i].FilePath == "" {
						(*target)[i].FilePath = saved.FilePath
					}
					(*target)[i].RetentionHoldUntil = holdUntil
					break
				}
			}
			if !found {
				*target = append(*target, saved)
			}
		}
	}
	record.Restored = true
	record.RestoredAt = &now
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return r.writeLocked(func(d *Data) error {
		restoreSongs(&d.DormSongs, record.Songs.DormSongs)
		restoreSongs(&d.BroadcastSongs, record.Songs.BroadcastSongs)
		if d.Weeks == nil {
			d.Weeks = make(map[string]map[string]models.WeekFile)
		}
		for key, index := range record.Weeks {
			if d.Weeks[key] == nil {
				d.Weeks[key] = make(map[string]models.WeekFile)
			}
			for number, file := range index {
				d.Weeks[key][number] = file
			}
		}
		return nil
	}, moves, map[string]json.RawMessage{recordName: data})
}
