package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"

	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
)

func setupSongHandler(t *testing.T) (*SongHandler, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := repo.New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New repo: %v", err)
	}
	songs := songstore.New(r)
	snapshots := snapshotstore.New(r, func(format string, args ...interface{}) { t.Logf(format, args...) })
	// Library 传 nil：这些用例不涉及音频文件，syncAfterChange 会跳过
	return NewSongHandler(songs, snapshots, nil), dir
}

func TestHandleCreateAndGet(t *testing.T) {
	h, _ := setupSongHandler(t)
	body, _ := json.Marshal(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/songs", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	var song models.Song
	if err := json.Unmarshal(rec.Body.Bytes(), &song); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if song.Title != "Test" || song.Weekday == "" {
		t.Fatalf("unexpected created song: %+v", song)
	}
}

func TestHandleList(t *testing.T) {
	h, _ := setupSongHandler(t)
	if _, err := h.Songs.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A"}); err != nil {
		t.Fatalf("AddSong: %v", err)
	}

	r := chi.NewRouter()
	r.Get("/", h.HandleList)
	req := httptest.NewRequest(http.MethodGet, "/?type=dorm", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var songs []models.Song
	if err := json.Unmarshal(rec.Body.Bytes(), &songs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(songs) != 1 {
		t.Fatalf("expected 1 song, got %d", len(songs))
	}
}

func uploadSongSheet(t *testing.T, h *SongHandler, titles ...string) *httptest.ResponseRecorder {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	sheet := book.GetSheetName(0)
	if err := book.SetSheetRow(sheet, "A1", &[]string{"日期", "星期", "歌名", "备注"}); err != nil {
		t.Fatal(err)
	}
	for i, title := range titles {
		if err := book.SetSheetRow(sheet, fmt.Sprintf("A%d", i+2), &[]string{"2026-07-25", "", title}); err != nil {
			t.Fatal(err)
		}
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "songs.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if err := book.Write(file); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/songs/import?type=dorm", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	rec := httptest.NewRecorder()
	h.HandleImport(rec, req)
	return rec
}

func TestHandleImportAssignsUniqueIDsAndBacksUpOriginalSongs(t *testing.T) {
	h, dir := setupSongHandler(t)
	old, err := h.Songs.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Previous song"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := h.Songs.AddSong(models.CreateSongRequest{Type: "broadcast", Date: "2026-07-25", Title: "Broadcast song"})
	if err != nil {
		t.Fatal(err)
	}
	rec := uploadSongSheet(t, h, "First song", "Second song", "Third song")
	if rec.Code != http.StatusOK {
		t.Fatalf("import failed: %d %s", rec.Code, rec.Body)
	}
	songs := h.Songs.GetDormSongs()
	if len(songs) != 3 {
		t.Fatalf("expected three imported songs, got %v", songs)
	}
	for i, song := range songs {
		if want := other.ID + 1 + int64(i); song.ID != want {
			t.Fatalf("song %q ID = %d, want %d", song.Title, song.ID, want)
		}
	}
	added, err := h.Songs.AddSong(models.CreateSongRequest{Type: "broadcast", Date: "2026-07-26", Title: "Added after import"})
	if err != nil || added.ID != songs[2].ID+1 {
		t.Fatalf("subsequent song ID = %d, error = %v", added.ID, err)
	}
	files, err := filepath.Glob(filepath.Join(paths.GetSnapshotsDir(dir), "snapshot-pre-import-*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("expected one pre-import snapshot, got %v (%v)", files, err)
	}
	if err := h.Snapshots.RestoreFromSnapshot(filepath.Base(files[0])); err != nil {
		t.Fatal(err)
	}
	restored := h.Songs.GetDormSongs()
	if len(restored) != 1 || restored[0].ID != old.ID || restored[0].Title != old.Title {
		t.Fatalf("restored unexpected songs: %v", restored)
	}
}

func TestHandleImportKeepsPlaylistWhenBackupFails(t *testing.T) {
	h, dir := setupSongHandler(t)
	old, err := h.Songs.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Previous song"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.GetSnapshotsDir(dir), []byte("blocks the snapshot directory"), 0600); err != nil {
		t.Fatal(err)
	}
	rec := uploadSongSheet(t, h, "Replacement song")
	var result models.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError || result.Error.Code != "PRE_IMPORT_SNAPSHOT_FAILED" {
		t.Fatalf("expected backup error, got %d %s", rec.Code, rec.Body)
	}
	r, err := repo.New(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, store := range []*songstore.SongStore{h.Songs, songstore.New(r)} {
		songs := store.GetDormSongs()
		if len(songs) != 1 || songs[0].ID != old.ID || songs[0].Title != old.Title {
			t.Fatalf("backup failure modified the playlist: %v", songs)
		}
	}
}
