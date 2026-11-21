package repo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"broadcast-tool/models"
)

func TestMigrateRetainRestoreAndExpire(t *testing.T) {
	dir := t.TempDir()
	oldAudio := "2026年第40周/01.mp3"
	freshAudio := "2026年第46周/01.mp3"
	write := func(name string, b []byte) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("MusicFiles/"+oldAudio, []byte("old recording"))
	write("MusicFiles/"+freshAudio, []byte("recent recording"))
	d := Data{
		DormSongs: []models.Song{
			{ID: 1, Date: "2026-10-01", FilePath: oldAudio, Title: "old recording"},
			{ID: 2, Date: "2026-11-12", FilePath: freshAudio, Title: "retained"},
			{ID: 3, Date: "2027-02-04", Title: "future"},
			{ID: 99, Date: "2023-11-10", Title: "expired metadata"},
		},
		Settings: DefaultSettings(),
		Weeks: map[string]map[string]models.WeekFile{
			"2026年第40周": {"01": {Kind: models.WeekFileSong, SongIDs: []int64{1}}},
			"2026年第46周": {"01": {Kind: models.WeekFileSong, SongIDs: []int64{2}}},
		},
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	write("data.json", b)
	r, err := New(dir, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, weekMetadata("2026年第40周"))); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.Local)
	if err := r.Collect(now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "MusicFiles", oldAudio)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old audio not retired: %v", err)
	}
	reopened, err := New(dir, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	reopened.Read(func(data *Data) {
		if len(data.DormSongs) != 3 || data.DormSongs[0].FilePath != "" || data.NextSongID != 100 {
			t.Fatalf("unexpected active records: %+v", data)
		}
	})
	list, err := reopened.Recoveries()
	if err != nil || len(list) != 1 || list[0].SongCount != 2 || list[0].FileCount != 1 {
		t.Fatalf("unexpected recycle state: %+v, %v", list, err)
	}
	if err := reopened.RestoreRecovery(list[0].ID, now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if content, err := os.ReadFile(filepath.Join(dir, "MusicFiles", oldAudio)); err != nil || string(content) != "old recording" {
		t.Fatalf("restore audio: %q %v", content, err)
	}
	if err := reopened.Collect(now.AddDate(0, 0, 2)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "MusicFiles", oldAudio)); err != nil {
		t.Fatal("temporary restore hold lost")
	}
	if err := reopened.Collect(now.AddDate(0, 0, 9)); err != nil {
		t.Fatal(err)
	}
	if err := reopened.RestoreRecovery(list[0].ID, now.AddDate(0, 0, 9)); err == nil {
		t.Fatal("expired recovery remained usable")
	}
}

func TestRecoverCommittedFileTransaction(t *testing.T) {
	r, dir := newTestRepo(t)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	header, err := root.ReadFile("data.json")
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := atomicFile(root, name, data); err != nil {
			t.Fatal(err)
		}
	}
	write("MusicFiles/_staging/track.mp3", []byte("audio"))
	txn := fileTransaction{
		Moves:  []FileMove{{"MusicFiles/_staging/track.mp3", "recycle/test/files/track.mp3"}},
		Writes: map[string]json.RawMessage{"data.json": header, archiveName: []byte(`{"dormSongs":[{"id":3,"title":"kept"}]}`)},
	}
	b, err := json.Marshal(txn)
	if err != nil {
		t.Fatal(err)
	}
	write(journalName, b)
	if err := root.MkdirAll("recycle/test/files", 0700); err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(txn.Moves[0].From, txn.Moves[0].To); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(func(*Data) error { return nil }); err == nil {
		t.Fatal("pending journal was overwritten")
	}
	restarted, err := New(dir, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	restarted.Read(func(data *Data) {
		if len(data.DormSongs) != 1 || data.DormSongs[0].Title != "kept" {
			t.Fatal("replay lost data")
		}
	})
	if _, err := root.Stat(journalName); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("journal not cleared")
	}
}

func TestCollectStrandedManagedAudio(t *testing.T) {
	r, dir := newTestRepo(t)
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.Local)
	old := now.AddDate(0, 0, -45)
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	weekFile := "MusicFiles/2026年第40周/03.mp3"
	stageFile := "MusicFiles/_staging/3815d148-fd86-4502-82a6-3a32af806746.mp3"
	manualFile := "MusicFiles/2026年第40周/manual.mp3"
	for _, file := range []string{weekFile, stageFile, manualFile} {
		if err := atomicFile(root, file, []byte("sample")); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filepath.Join(dir, file), old, old); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Collect(now); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{weekFile, stageFile} {
		if _, err := root.Stat(file); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("uncollected file %s: %v", file, err)
		}
	}
	if _, err := root.Stat(manualFile); err != nil {
		t.Fatal(err)
	}
	entries, err := r.Recoveries()
	if err != nil || len(entries) != 1 || entries[0].FileCount != 2 {
		t.Fatalf("quarantine: %+v, %v", entries, err)
	}
	if err := r.RestoreRecovery(entries[0].ID, now.AddDate(0, 0, 6)); err != nil {
		t.Fatal(err)
	}
	if err := r.Collect(now.AddDate(0, 0, 8)); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{weekFile, stageFile} {
		if _, err := root.Stat(file); err != nil {
			t.Fatalf("restored file %s: %v", file, err)
		}
	}
}
