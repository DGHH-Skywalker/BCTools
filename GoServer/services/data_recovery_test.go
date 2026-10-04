package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"
)

func TestRecoverConflictingRecordPreservesCurrentAndExtractsMergedAudio(t *testing.T) {
	root, old := t.TempDir(), t.TempDir()
	r, err := repo.New(root, func(string, ...interface{}) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Write(func(d *repo.Data) error {
		d.BroadcastSongs = []models.Song{{ID: 1, Title: "现有", Date: "2026-10-04", Period: "noon"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	source := repo.Data{DormSongs: []models.Song{}, BroadcastSongs: []models.Song{{ID: 1, Title: "旧目录本周", Date: "2026-10-04", FilePath: "2026年第40周/05.mp3"}}, Weeks: map[string]map[string]models.WeekFile{"2026年第40周": {"05": {Kind: models.WeekFileMerged, Parts: []models.MergedPart{{SongID: 1, Offset: 1, Length: 3}}}}}}
	b, _ := json.Marshal(source)
	os.WriteFile(filepath.Join(old, "data.json"), b, 0600)
	audio := filepath.Join(paths.GetMusicFilesDir(old), "2026年第40周", "05.mp3")
	os.MkdirAll(filepath.Dir(audio), 0700)
	os.WriteFile(audio, []byte("XabcDEF"), 0600)
	if err := RecordDataSources(root, []string{old}); err != nil {
		t.Fatal(err)
	}
	info := InspectDataSource(old)
	loadedInfo, data, err := LoadDataSource(root, info.ID, info.Digest)
	if err != nil {
		t.Fatal(err)
	}
	count, err := RecoverSelected(r, loadedInfo, data, []RecoverySelection{{Type: "broadcast", ID: 1}})
	if err != nil || count != 1 {
		t.Fatalf("recover: %d %v", count, err)
	}
	var songs []models.Song
	r.Read(func(d *repo.Data) { songs = append(songs, d.BroadcastSongs...) })
	if len(songs) != 2 || songs[0].Title != "现有" || songs[1].ID == 1 || songs[1].Period != "" {
		t.Fatal(songs)
	}
	actual, err := os.ReadFile(filepath.Join(paths.GetSongStagingDir(root), songs[1].FilePath))
	if err != nil || !bytes.Equal(actual, []byte("abc")) {
		t.Fatalf("wrong recovered audio: %q %v", actual, err)
	}
	before, _ := os.ReadFile(audio)
	if !bytes.Equal(before, []byte("XabcDEF")) {
		t.Fatal("modified source audio")
	}
	backups, _ := os.ReadDir(paths.GetSnapshotsDir(root))
	if len(backups) != 1 {
		t.Fatal("missing pre-import backup")
	}
	os.WriteFile(filepath.Join(old, "data.json"), []byte(`{"dormSongs":[],"broadcastSongs":[]}`), 0600)
	if _, _, err := LoadDataSource(root, info.ID, info.Digest); err == nil {
		t.Fatal("accepted a changed source")
	}
}
