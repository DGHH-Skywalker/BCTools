package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"broadcast-tool/converter"
	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"
)

func newTestLibrary(t *testing.T) (*MusicLibrary, *repo.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	r, err := repo.New(dir, func(format string, args ...interface{}) { t.Logf(format, args...) })
	if err != nil {
		t.Fatalf("New repo: %v", err)
	}
	// 测试不触发 ffmpeg：用空路径构造，静音生成一律走 silentTemplate 复制
	conv := converter.New("", "")
	lib := NewMusicLibrary(dir, r, conv, func(format string, args ...interface{}) { t.Logf(format, args...) })
	return lib, r, dir
}

// fakeMP3 构造带 ID3v2 头与 ID3v1 尾的伪 MP3（帧区是任意字节）。
func fakeMP3(t *testing.T, tag, frames string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if tag != "" {
		// 'ID3' ver flags size(4, synchsafe) + tag 内容
		buf.WriteString("ID3")
		buf.Write([]byte{3, 0, 0})
		n := len(tag)
		buf.Write([]byte{byte(n >> 21), byte(n >> 14), byte(n >> 7), byte(n & 0x7F)})
		buf.WriteString(tag)
	}
	buf.WriteString(frames)
	buf.WriteString("TAG" + strings.Repeat("x", 125))
	return buf.Bytes()
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func setSlots(t *testing.T, r *repo.Repository, slots ...models.TimeSlot) {
	t.Helper()
	if err := r.Write(func(d *repo.Data) error {
		d.Settings.TimeSlots = slots
		return nil
	}); err != nil {
		t.Fatalf("set slots: %v", err)
	}
}

func addDormSong(t *testing.T, r *repo.Repository, id int64, date, slotID, filePath string, order int) {
	t.Helper()
	if err := r.Write(func(d *repo.Data) error {
		slot := slotID
		d.DormSongs = append(d.DormSongs, models.Song{
			ID: id, Date: date, Title: fmt.Sprintf("song-%d", id),
			FilePath: filePath, TimeSlotID: &slot, Order: order,
		})
		return nil
	}); err != nil {
		t.Fatalf("add song: %v", err)
	}
}

func getSong(t *testing.T, r *repo.Repository, id int64) models.Song {
	t.Helper()
	var out models.Song
	r.Read(func(d *repo.Data) {
		for _, s := range d.DormSongs {
			if s.ID == id {
				out = s
			}
		}
	})
	return out
}

func TestWeekNameAndDates(t *testing.T) {
	name, err := WeekNameForDate("2026-08-19")
	if err != nil {
		t.Fatalf("WeekNameForDate: %v", err)
	}
	if name != "2026年第34周" {
		t.Fatalf("expected 2026年第34周, got %s", name)
	}
	dates, err := weekDates("2026年第34周")
	if err != nil {
		t.Fatalf("weekDates: %v", err)
	}
	if len(dates) != 7 || dates[0] != "2026-08-17" || dates[6] != "2026-08-23" {
		t.Fatalf("unexpected dates: %v", dates)
	}
	y, w, err := ParseWeekName("2026年第34周")
	if err != nil || y != 2026 || w != 34 {
		t.Fatalf("ParseWeekName: %v %d %d", err, y, w)
	}
	if isoWeekday("2026-08-17") != 1 || isoWeekday("2026-08-23") != 7 {
		t.Fatalf("isoWeekday mismatch")
	}
}

func TestSplitID3Tags(t *testing.T) {
	data := fakeMP3(t, "tagdata", "FRAMEFRAME")
	v2, v1, frames := converter.SplitID3Tags(data)
	if string(v2[:3]) != "ID3" || !bytes.Contains(v2, []byte("tagdata")) {
		t.Fatalf("v2 tag wrong: %x", v2)
	}
	if string(v1[:3]) != "TAG" || len(v1) != 128 {
		t.Fatalf("v1 tag wrong")
	}
	if string(frames) != "FRAMEFRAME" {
		t.Fatalf("frames wrong: %q", frames)
	}
}

// 场景：周一两个时段，时段 1 一首歌，时段 2 两首歌（合并）。
// 同步后 01=单曲、02=合并；合并成员可无损还原。
func TestSyncWeekPlaceMergeAndExtract(t *testing.T) {
	lib, r, root := newTestLibrary(t)
	setSlots(t, r,
		models.TimeSlot{ID: "s1", DayIndex: 1, Order: 1, Time: "06:20"},
		models.TimeSlot{ID: "s2", DayIndex: 1, Order: 2, Time: "06:35"},
	)

	staging := paths.GetSongStagingDir(root)
	src1 := fakeMP3(t, "tag-one", "AAAA")
	src2 := fakeMP3(t, "tag-two", "BBBB")
	src3 := fakeMP3(t, "tag-three", "CCCC")
	writeFile(t, filepath.Join(staging, "uuid-1.mp3"), src1)
	writeFile(t, filepath.Join(staging, "uuid-2.mp3"), src2)
	writeFile(t, filepath.Join(staging, "uuid-3.mp3"), src3)

	addDormSong(t, r, 1, "2026-08-17", "s1", "uuid-1.mp3", 0)
	addDormSong(t, r, 2, "2026-08-17", "s2", "uuid-2.mp3", 0)
	addDormSong(t, r, 3, "2026-08-17", "s2", "uuid-3.mp3", 1)

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek: %v", err)
	}

	weekDir := filepath.Join(paths.GetMusicFilesDir(root), "2026年第34周")

	// 01 = song 1（完整文件，含原标签）
	got1, err := os.ReadFile(filepath.Join(weekDir, "01.mp3"))
	if err != nil {
		t.Fatalf("read 01.mp3: %v", err)
	}
	if !bytes.Equal(got1, src1) {
		t.Fatalf("01.mp3 should equal source of song 1")
	}

	// 索引与 FilePath
	var wf map[string]models.WeekFile
	r.Read(func(d *repo.Data) { wf = d.Weeks["2026年第34周"] })
	if wf["01"].Kind != models.WeekFileSong || len(wf["01"].SongIDs) != 1 || wf["01"].SongIDs[0] != 1 {
		t.Fatalf("index 01 wrong: %+v", wf["01"])
	}
	if wf["02"].Kind != models.WeekFileMerged || len(wf["02"].SongIDs) != 2 {
		t.Fatalf("index 02 wrong: %+v", wf["02"])
	}
	if got := getSong(t, r, 1).FilePath; got != "2026年第34周/01.mp3" {
		t.Fatalf("song 1 FilePath = %q", got)
	}
	if got := getSong(t, r, 3).FilePath; got != "2026年第34周/02.mp3" {
		t.Fatalf("song 3 FilePath = %q", got)
	}

	// 合并成员 3 可以无损还原
	for _, p := range wf["02"].Parts {
		if p.SongID == 3 {
			tmp, err := lib.extractPart(filepath.Join(weekDir, "02.mp3"), p)
			if err != nil {
				t.Fatalf("extractPart: %v", err)
			}
			defer os.Remove(tmp)
			got, _ := os.ReadFile(tmp)
			if !bytes.Equal(got, src3) {
				t.Fatalf("extracted part of song 3 differs from original")
			}
		}
	}

	// 暂存区应被清空（文件已入库）
	entries, _ := os.ReadDir(staging)
	if len(entries) != 0 {
		t.Fatalf("staging should be empty, got %d files", len(entries))
	}
}

// 场景：在更早的位置插入一个已填歌的时段，后续编号整体后移。
// 同步应只重建受影响的编号，且文件内容不丢。
func TestSyncWeekRenumber(t *testing.T) {
	lib, r, root := newTestLibrary(t)
	setSlots(t, r,
		models.TimeSlot{ID: "s1", DayIndex: 1, Order: 1, Time: "06:20"},
	)
	staging := paths.GetSongStagingDir(root)
	src1 := fakeMP3(t, "tag-one", "AAAA")
	writeFile(t, filepath.Join(staging, "uuid-1.mp3"), src1)
	addDormSong(t, r, 1, "2026-08-17", "s1", "uuid-1.mp3", 0)

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek: %v", err)
	}

	// 插入更早的时段 s0（也有歌）
	setSlots(t, r,
		models.TimeSlot{ID: "s0", DayIndex: 1, Order: 0, Time: "06:10"},
		models.TimeSlot{ID: "s1", DayIndex: 1, Order: 1, Time: "06:20"},
	)
	src0 := fakeMP3(t, "tag-zero", "ZZZZ")
	writeFile(t, filepath.Join(staging, "uuid-0.mp3"), src0)
	addDormSong(t, r, 0, "2026-08-17", "s0", "uuid-0.mp3", 0)

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek 2: %v", err)
	}

	weekDir := filepath.Join(paths.GetMusicFilesDir(root), "2026年第34周")
	got0, _ := os.ReadFile(filepath.Join(weekDir, "01.mp3"))
	got1, _ := os.ReadFile(filepath.Join(weekDir, "02.mp3"))
	if !bytes.Equal(got0, src0) {
		t.Fatalf("01.mp3 should be song 0")
	}
	if !bytes.Equal(got1, src1) {
		t.Fatalf("02.mp3 should be song 1")
	}
	if got := getSong(t, r, 1).FilePath; got != "2026年第34周/02.mp3" {
		t.Fatalf("song 1 FilePath = %q", got)
	}
}

// 场景：歌曲日期改到别的周，旧周同步时音频应搬到暂存区而不是丢失。
func TestSyncWeekRelocation(t *testing.T) {
	lib, r, root := newTestLibrary(t)
	setSlots(t, r,
		models.TimeSlot{ID: "s1", DayIndex: 1, Order: 1, Time: "06:20"},
	)
	staging := paths.GetSongStagingDir(root)
	src1 := fakeMP3(t, "tag-one", "AAAA")
	writeFile(t, filepath.Join(staging, "uuid-1.mp3"), src1)
	addDormSong(t, r, 1, "2026-08-17", "s1", "uuid-1.mp3", 0)

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek: %v", err)
	}

	// 预置一个静音模板（测试环境不跑 ffmpeg）
	weekDir := filepath.Join(paths.GetMusicFilesDir(root), "2026年第34周")
	writeFile(t, filepath.Join(weekDir, "99.mp3"), fakeMP3(t, "", "SILENT"))
	r.Write(func(d *repo.Data) error {
		if d.Weeks == nil {
			d.Weeks = map[string]map[string]models.WeekFile{}
		}
		if d.Weeks["2026年第34周"] == nil {
			d.Weeks["2026年第34周"] = map[string]models.WeekFile{}
		}
		d.Weeks["2026年第34周"]["99"] = models.WeekFile{Kind: models.WeekFileSilent}
		return nil
	})

	// 日期改到下一周（2026年第35周），旧时段空出
	r.Write(func(d *repo.Data) error {
		d.DormSongs[0].Date = "2026-08-24"
		return nil
	})

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek 2: %v", err)
	}

	song := getSong(t, r, 1)
	if song.FilePath != "song-1.mp3" {
		t.Fatalf("song 1 should be relocated to staging, got %q", song.FilePath)
	}
	got, err := os.ReadFile(filepath.Join(staging, "song-1.mp3"))
	if err != nil {
		t.Fatalf("read relocated file: %v", err)
	}
	if !bytes.Equal(got, src1) {
		t.Fatalf("relocated file content differs")
	}

	// 同步下一周应把它重新入库
	if err := lib.SyncWeek("2026年第35周"); err != nil {
		t.Fatalf("SyncWeek 35: %v", err)
	}
	if got := getSong(t, r, 1).FilePath; got != "2026年第35周/01.mp3" {
		t.Fatalf("song 1 FilePath = %q", got)
	}
}

// 场景：删除合并文件中的一首歌，剩余成员重组后仍能还原。
func TestSyncWeekRemoveMergedMember(t *testing.T) {
	lib, r, root := newTestLibrary(t)
	setSlots(t, r,
		models.TimeSlot{ID: "s2", DayIndex: 1, Order: 2, Time: "06:35"},
	)
	staging := paths.GetSongStagingDir(root)
	src2 := fakeMP3(t, "tag-two", "BBBB")
	src3 := fakeMP3(t, "tag-three", "CCCC")
	writeFile(t, filepath.Join(staging, "uuid-2.mp3"), src2)
	writeFile(t, filepath.Join(staging, "uuid-3.mp3"), src3)
	addDormSong(t, r, 2, "2026-08-17", "s2", "uuid-2.mp3", 0)
	addDormSong(t, r, 3, "2026-08-17", "s2", "uuid-3.mp3", 1)

	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek: %v", err)
	}

	// 删除 song 2，剩 song 3 单独成曲
	r.Write(func(d *repo.Data) error {
		for i, s := range d.DormSongs {
			if s.ID == 2 {
				d.DormSongs = append(d.DormSongs[:i], d.DormSongs[i+1:]...)
				break
			}
		}
		return nil
	})
	if err := lib.SyncWeek("2026年第34周"); err != nil {
		t.Fatalf("SyncWeek 2: %v", err)
	}

	weekDir := filepath.Join(paths.GetMusicFilesDir(root), "2026年第34周")
	got, err := os.ReadFile(filepath.Join(weekDir, "01.mp3"))
	if err != nil {
		t.Fatalf("read 01.mp3: %v", err)
	}
	if !bytes.Equal(got, src3) {
		t.Fatalf("01.mp3 should be song 3's original file after un-merge")
	}
}
