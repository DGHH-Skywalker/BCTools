package services

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"
	"github.com/google/uuid"
)

type RecoveryRecord struct {
	Type      string       `json:"type"`
	Song      models.Song  `json:"song"`
	Existing  *models.Song `json:"existing,omitempty"`
	Identical bool         `json:"identical"`
}
type RecoverySelection struct {
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

func LoadDataSource(root, id, digest string) (DataSource, repo.Data, error) {
	sources, err := ListDataSources(root)
	if err != nil {
		return DataSource{}, repo.Data{}, err
	}
	for _, source := range sources {
		if source.ID != id {
			continue
		}
		if source.Error != "" {
			return source, repo.Data{}, fmt.Errorf("%s", source.Error)
		}
		b, err := os.ReadFile(paths.GetDataFilePath(source.Directory))
		if err != nil {
			return source, repo.Data{}, err
		}
		actualDigest := fmt.Sprintf("%x", sha256.Sum256(b))
		if digest != "" && digest != actualDigest {
			return source, repo.Data{}, fmt.Errorf("来源已发生变化，请重新比对")
		}
		data, err := decodeDataSource(b)
		source.Digest = actualDigest
		return source, data, err
	}
	return DataSource{}, repo.Data{}, fmt.Errorf("未登记的数据来源")
}

func RecoveryPreview(current, source repo.Data) []RecoveryRecord {
	result := []RecoveryRecord{}
	for _, kind := range []string{"dorm", "broadcast"} {
		incoming, existing := source.DormSongs, current.DormSongs
		if kind == "broadcast" {
			incoming, existing = source.BroadcastSongs, current.BroadcastSongs
		}
		for _, song := range incoming {
			record := RecoveryRecord{Type: kind, Song: song}
			for _, old := range existing {
				if old.ID == song.ID {
					old.Type = ""
					old.Weekday = ""
					song.Type = ""
					song.Weekday = ""
					record.Existing = &old
					record.Identical = reflect.DeepEqual(old, song)
					break
				}
			}
			result = append(result, record)
		}
	}
	return result
}

// Import selected records as NEW identities. Both versions of a conflicting
// record survive. Audio uses unique staged names; source resources are untouched.
func RecoverSelected(r *repo.Repository, sourceInfo DataSource, source repo.Data, selected []RecoverySelection) (int, error) {
	if len(selected) == 0 {
		return 0, fmt.Errorf("请先选择要补回的歌单记录")
	}
	currentBytes, err := r.SnapshotData()
	if err != nil {
		return 0, err
	}
	backup := filepath.Join(paths.GetSnapshotsDir(r.AppDataDir()), "pre-recovery-"+time.Now().Format("20060102-150405")+"-"+uuid.NewString()+".json")
	if err := os.MkdirAll(filepath.Dir(backup), 0700); err != nil {
		return 0, err
	}
	if err := atomicBytes(backup, currentBytes); err != nil {
		return 0, err
	}
	var current repo.Data
	if err = json.Unmarshal(currentBytes, &current); err != nil {
		return 0, err
	}
	preview := RecoveryPreview(current, source)
	records := []RecoveryRecord{}
	seen := map[string]bool{}
	for _, selection := range selected {
		key := fmt.Sprintf("%s:%d", selection.Type, selection.ID)
		if seen[key] {
			continue
		}
		seen[key] = true
		found := false
		for _, record := range preview {
			if record.Type == selection.Type && record.Song.ID == selection.ID {
				found = true
				if !record.Identical {
					records = append(records, record)
				}
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("来源中找不到所选记录：%s", key)
		}
	}
	for i := range records {
		song := &records[i].Song
		if song.FilePath != "" {
			path, err := sourceAudioPath(sourceInfo.Directory, song.FilePath)
			if err != nil {
				return 0, err
			}
			parts := strings.SplitN(filepath.ToSlash(strings.ReplaceAll(song.FilePath, "\\", "/")), "/", 2)
			if len(parts) == 2 {
				wf := source.Weeks[parts[0]][strings.TrimSuffix(parts[1], ".mp3")]
				if wf.Kind == models.WeekFileMerged {
					matched := false
					for _, part := range wf.Parts {
						if part.SongID == song.ID {
							lib := &MusicLibrary{DataRoot: r.AppDataDir()}
							path, err = lib.extractPart(path, part)
							if err != nil {
								return 0, err
							}
							defer os.Remove(path)
							matched = true
							break
						}
					}
					if !matched {
						return 0, fmt.Errorf("合并音频缺少歌曲 %d 的成员信息", song.ID)
					}
				}
			}
			name := uuid.NewString() + filepath.Ext(path)
			destination := filepath.Join(paths.GetSongStagingDir(r.AppDataDir()), name)
			if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return 0, err
			}
			if err := copyFileAtomic(path, destination); err != nil {
				return 0, err
			}
			song.FilePath = name
		}
	}
	count := 0
	err = r.Write(func(data *repo.Data) error {
		var maxID int64
		for _, list := range [][]models.Song{data.DormSongs, data.BroadcastSongs} {
			for _, song := range list {
				if song.ID > maxID {
					maxID = song.ID
				}
			}
		}
		slotMapping := map[string]string{}
		usedSlots := map[string]bool{}
		for _, record := range records {
			if record.Song.TimeSlotID != nil {
				usedSlots[*record.Song.TimeSlotID] = true
			}
		}
		for _, old := range source.Settings.TimeSlots {
			if !usedSlots[old.ID] {
				continue
			}
			matches := []string{}
			for _, slot := range data.Settings.TimeSlots {
				if slot.DayIndex == old.DayIndex && slot.Time == old.Time {
					matches = append(matches, slot.ID)
				}
			}
			if len(matches) == 1 {
				slotMapping[old.ID] = matches[0]
			} else {
				fresh := old
				fresh.ID = uuid.NewString()
				data.Settings.TimeSlots = append(data.Settings.TimeSlots, fresh)
				slotMapping[old.ID] = fresh.ID
			}
		}
		for _, record := range records {
			// Recheck inside the transaction: another client may have edited a
			// record after the preview. No existing record is ever overwritten.
			song := record.Song
			maxID++
			song.ID = maxID
			song.Type = ""
			song.Weekday = ""
			if song.TimeSlotID != nil {
				if id := slotMapping[*song.TimeSlotID]; id != "" {
					song.TimeSlotID = &id
				} else {
					song.TimeSlotID = nil
				}
			}
			if record.Type == "dorm" {
				data.DormSongs = append(data.DormSongs, song)
			} else {
				data.BroadcastSongs = append(data.BroadcastSongs, song)
			}
			count++
		}
		return nil
	})
	return count, err
}
