package songstore

import (
	"strings"
	"testing"

	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
)

func newTestSongStore(t *testing.T) (*SongStore, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := repo.New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New repo: %v", err)
	}
	return New(r), dir
}

func TestSongCRUD(t *testing.T) {
	s, _ := newTestSongStore(t)
	song, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Test", Artist: "Artist"})
	if err != nil {
		t.Fatalf("AddSong: %v", err)
	}
	if song.ID != 1 {
		t.Fatalf("expected ID 1, got %d", song.ID)
	}
	if song.Weekday == "" {
		t.Fatalf("expected computed weekday in response")
	}

	songs := s.GetSongsByType("dorm", nil)
	if len(songs) != 1 {
		t.Fatalf("expected 1 dorm song, got %d", len(songs))
	}

	newTitle := "Updated"
	updated, err := s.UpdateSong(song.ID, models.UpdateSongRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateSong: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected updated title")
	}

	if err := s.DeleteSong(song.ID); err != nil {
		t.Fatalf("DeleteSong: %v", err)
	}
	if len(s.GetSongsByType("dorm", nil)) != 0 {
		t.Fatalf("expected 0 songs after delete")
	}
}

// 同一宿舍时段允许放置多首歌曲：向同一时段重复添加不应报错，且两首都应存在。
func TestDormSlotMultipleSongs(t *testing.T) {
	s, _ := newTestSongStore(t)
	slotID := "slot-1"
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A", TimeSlotID: &slotID}); err != nil {
		t.Fatalf("AddSong A: %v", err)
	}
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "B", TimeSlotID: &slotID}); err != nil {
		t.Fatalf("AddSong B to same slot should succeed, got: %v", err)
	}
	songs := s.GetSongsByType("dorm", nil)
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs in the same slot, got %d", len(songs))
	}
}

// 宿舍歌单按周卡片网格管理时，「添加歌曲」会先创建空标题占位行，由用户在内联输入框中填写。
// 因此 dorm 类型应允许创建空标题（与 broadcast 一致）。
func TestDormEmptyTitleAllowed(t *testing.T) {
	s, _ := newTestSongStore(t)
	song, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: ""})
	if err != nil {
		t.Fatalf("AddSong with empty dorm title should succeed, got: %v", err)
	}
	if song.Title != "" {
		t.Fatalf("expected empty title, got %q", song.Title)
	}
}

func TestCheckDuplicate(t *testing.T) {
	s, _ := newTestSongStore(t)
	if _, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A"}); err != nil {
		t.Fatalf("AddSong: %v", err)
	}
	if !s.CheckDuplicate("dorm", "2026-07-25", "  a  ") {
		t.Fatalf("expected duplicate")
	}
	if s.CheckDuplicate("dorm", "2026-07-26", "A") {
		t.Fatalf("expected no duplicate")
	}
}

func TestSortDormSongs(t *testing.T) {
	s, _ := newTestSongStore(t)
	var slots []models.TimeSlot
	s.repo.Read(func(data *repo.Data) {
		slots = data.Settings.TimeSlots
	})
	if len(slots) < 2 {
		t.Fatalf("expected at least 2 default slots")
	}
	_, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-26", Title: "Later", TimeSlotID: &slots[1].ID})
	if err != nil {
		t.Fatalf("AddSong: %v", err)
	}
	_, err = s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "Earlier", TimeSlotID: &slots[0].ID})
	if err != nil {
		t.Fatalf("AddSong: %v", err)
	}
	if err := s.SortDormSongs(); err != nil {
		t.Fatalf("SortDormSongs: %v", err)
	}
	songs := s.GetDormSongs()
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}
	if songs[0].Date > songs[1].Date {
		t.Fatalf("songs not sorted by date")
	}
}

// dormSlotTitles returns the comma-joined titles of dorm songs in a given
// date+slot, in the order GetSongsByType serves them (canonical order).
func dormSlotTitles(s *SongStore, date, slotID string) string {
	var parts []string
	for _, song := range s.GetSongsByType("dorm", nil) {
		if song.Date == date && song.TimeSlotID != nil && *song.TimeSlotID == slotID {
			parts = append(parts, song.Title)
		}
	}
	return strings.Join(parts, ",")
}

// 同一时段多首歌的拖拽排序：ReorderSongs 后，GetSongsByType 必须按新的拖拽顺序返回。
func TestReorderSongsWithinSlot(t *testing.T) {
	s, _ := newTestSongStore(t)
	var slots []models.TimeSlot
	s.repo.Read(func(data *repo.Data) { slots = data.Settings.TimeSlots })
	if len(slots) < 1 {
		t.Fatalf("expected at least 1 default slot")
	}
	slotID := slots[0].ID

	a, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A", TimeSlotID: &slotID})
	if err != nil {
		t.Fatalf("AddSong A: %v", err)
	}
	b, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "B", TimeSlotID: &slotID})
	if err != nil {
		t.Fatalf("AddSong B: %v", err)
	}
	c, err := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "C", TimeSlotID: &slotID})
	if err != nil {
		t.Fatalf("AddSong C: %v", err)
	}

	// 新歌追加到时段末尾，初始顺序为创建顺序 A,B,C。
	if got := dormSlotTitles(s, "2026-07-25", slotID); got != "A,B,C" {
		t.Fatalf("initial order = %q, want A,B,C", got)
	}

	// 拖拽为 C,A,B。
	if err := s.ReorderSongs([]int64{c.ID, a.ID, b.ID}); err != nil {
		t.Fatalf("ReorderSongs: %v", err)
	}

	if got := dormSlotTitles(s, "2026-07-25", slotID); got != "C,A,B" {
		t.Fatalf("after reorder = %q, want C,A,B", got)
	}
}

// 拖拽排序只影响给定 ID：其他时段的歌曲顺序应保持不变。
func TestReorderSongsIsolatedToSlot(t *testing.T) {
	s, _ := newTestSongStore(t)
	var slots []models.TimeSlot
	s.repo.Read(func(data *repo.Data) { slots = data.Settings.TimeSlots })
	if len(slots) < 2 {
		t.Fatalf("expected at least 2 default slots")
	}
	slotA, slotB := slots[0].ID, slots[1].ID

	a, _ := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "A", TimeSlotID: &slotA})
	b, _ := s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-25", Title: "B", TimeSlotID: &slotA})
	s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-26", Title: "X", TimeSlotID: &slotB})
	s.AddSong(models.CreateSongRequest{Type: "dorm", Date: "2026-07-26", Title: "Y", TimeSlotID: &slotB})

	// 只重排 slotA 为 B,A；slotB 不在传入 ID 中，应保持 X,Y。
	if err := s.ReorderSongs([]int64{b.ID, a.ID}); err != nil {
		t.Fatalf("ReorderSongs: %v", err)
	}

	if got := dormSlotTitles(s, "2026-07-25", slotA); got != "B,A" {
		t.Fatalf("slotA after reorder = %q, want B,A", got)
	}
	if got := dormSlotTitles(s, "2026-07-26", slotB); got != "X,Y" {
		t.Fatalf("slotB should be untouched = %q, want X,Y", got)
	}
}
