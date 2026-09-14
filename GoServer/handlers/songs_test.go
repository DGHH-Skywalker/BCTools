package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"

	"github.com/go-chi/chi/v5"
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
