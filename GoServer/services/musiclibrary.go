// Package services contains higher-level business logic that is shared by
// multiple handlers or too large to live directly in a handler.
package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"broadcast-tool/converter"
	"broadcast-tool/internal/repo"
	"broadcast-tool/models"
	"broadcast-tool/paths"
)

// MusicLibrary 维护 BctoolData\MusicFiles\<YYYY年第N周>\NN.mp3 的周文件夹布局。
//
// 周文件夹是「SD 卡最终形态」：编号锚定时段位置（与前端 exportEntries.ts
// 同一套规则），空时段是 17.43s 静音占位（标签写「空音频」），同时段多首歌
// 合并成一个 MP3。导出到桌面 = 整个周文件夹原样拷贝。
//
// data.json 的 weeks 字段就是索引：编号 -> 内容（哪些歌 / 是否合并 / 每个成员
// 的字节区间与原始 ID3 标签）。合并文件因此可以无损拆回单首歌。
//
// 一切变更（增删歌、改日期/时段、改时段配置）都通过 SyncWeek 收敛到期望布局，
// 按签名比对只重建发生变化的编号。
type MusicLibrary struct {
	mediaMu   sync.RWMutex
	taskMu    sync.Mutex
	tasks     sync.WaitGroup
	stopping  bool
	DataRoot  string
	Repo      *repo.Repository
	Converter *converter.FFMpegConverter
	Logger    func(format string, args ...interface{})
}

func (l *MusicLibrary) ScheduleSync(run func()) {
	l.taskMu.Lock()
	defer l.taskMu.Unlock()
	if l.stopping {
		return
	}
	l.tasks.Go(run)
}

func (l *MusicLibrary) WaitForSync() {
	l.taskMu.Lock()
	l.stopping = true
	l.taskMu.Unlock()
	l.tasks.Wait()
}

func NewMusicLibrary(
	dataRoot string,
	r *repo.Repository,
	c *converter.FFMpegConverter,
	logger func(format string, args ...interface{}),
) *MusicLibrary {
	if logger == nil {
		logger = func(format string, args ...interface{}) {}
	}
	return &MusicLibrary{DataRoot: dataRoot, Repo: r, Converter: c, Logger: logger}
}

// ---------------------------------------------------------------------------
// 周名与日期
// ---------------------------------------------------------------------------

// WeekNameForDate 把 "2026-08-19" 映射为 "2026年第34周"（ISO 周）。
func WeekNameForDate(dateStr string) (string, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date %q", dateStr)
	}
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d年第%d周", y, w), nil
}

// ParseWeekName 解析 "2026年第34周" -> (2026, 34)。
func ParseWeekName(weekName string) (int, int, error) {
	var year, week int
	if _, err := fmt.Sscanf(weekName, "%d年第%d周", &year, &week); err != nil {
		return 0, 0, fmt.Errorf("invalid week name %q", weekName)
	}
	if year < 1970 || year > 2999 || week < 1 || week > 53 {
		return 0, 0, fmt.Errorf("invalid week name %q", weekName)
	}
	return year, week, nil
}

// weekDates 返回某 ISO 周的 7 个日期（周一 ~ 周日）。
// 1 月 4 日恒在第 1 周，以它为锚点推算周一。
func weekDates(weekName string) ([]string, error) {
	year, week, err := ParseWeekName(weekName)
	if err != nil {
		return nil, err
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	isoWd := int(jan4.Weekday())
	if isoWd == 0 {
		isoWd = 7
	}
	monday := jan4.AddDate(0, 0, -(isoWd-1)+(week-1)*7)
	dates := make([]string, 7)
	for i := 0; i < 7; i++ {
		dates[i] = monday.AddDate(0, 0, i).Format("2006-01-02")
	}
	return dates, nil
}

// isoWeekday 返回日期的 ISO 星期（周一=1 … 周日=7），与 TimeSlot.DayIndex 对齐。
func isoWeekday(dateStr string) int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	d := int(t.Weekday())
	if d == 0 {
		d = 7
	}
	return d
}

// ---------------------------------------------------------------------------
// 期望布局
// ---------------------------------------------------------------------------

// weekSlot 是期望布局里「一个时段 = 一个编号」的描述。
type weekSlot struct {
	Num    string        // "05"
	Date   string        // 该时段属于周内的哪一天
	SlotID string        // TimeSlot.ID
	Songs  []models.Song // 该时段的歌（按播放顺序），只含有音频文件的
}

// computeLayout 计算 weekName 的期望布局：遍历周一~周日的全部时段，按顺序编号。
// 规则与 Web/src/utils/exportEntries.ts 一致，必须保持同步修改。
func (l *MusicLibrary) computeLayout(weekName string) ([]weekSlot, error) {
	dates, err := weekDates(weekName)
	if err != nil {
		return nil, err
	}

	var slots []models.TimeSlot
	var songs []models.Song
	l.Repo.Read(func(d *repo.Data) {
		slots = d.Settings.TimeSlots
		songs = d.DormSongs
	})

	byKey := make(map[string][]models.Song)
	for _, s := range songs {
		if s.TimeSlotID == nil || s.FilePath == "" {
			continue
		}
		if name, err := WeekNameForDate(s.Date); err != nil || name != weekName {
			continue
		}
		key := s.Date + "|" + *s.TimeSlotID
		byKey[key] = append(byKey[key], s)
	}

	var out []weekSlot
	seq := 1
	for _, date := range dates {
		dayIdx := isoWeekday(date)
		var daySlots []models.TimeSlot
		for _, slot := range slots {
			if slot.DayIndex == dayIdx {
				daySlots = append(daySlots, slot)
			}
		}
		sort.SliceStable(daySlots, func(i, j int) bool { return daySlots[i].Order < daySlots[j].Order })
		for _, slot := range daySlots {
			list := byKey[date+"|"+slot.ID]
			sort.SliceStable(list, func(i, j int) bool {
				if list[i].Order != list[j].Order {
					return list[i].Order < list[j].Order
				}
				return list[i].ID < list[j].ID
			})
			out = append(out, weekSlot{
				Num:    fmt.Sprintf("%02d", seq),
				Date:   date,
				SlotID: slot.ID,
				Songs:  list,
			})
			seq++
		}
	}
	return out, nil
}

// desiredFile 由时段内容推导该编号文件的期望元数据。
func desiredFile(slot weekSlot) models.WeekFile {
	switch {
	case len(slot.Songs) == 0:
		return models.WeekFile{Kind: models.WeekFileSilent}
	case len(slot.Songs) == 1:
		return models.WeekFile{Kind: models.WeekFileSong, SongIDs: []int64{slot.Songs[0].ID}}
	default:
		ids := make([]int64, 0, len(slot.Songs))
		for _, s := range slot.Songs {
			ids = append(ids, s.ID)
		}
		return models.WeekFile{Kind: models.WeekFileMerged, SongIDs: ids}
	}
}

// ---------------------------------------------------------------------------
// 路径解析
// ---------------------------------------------------------------------------

func (l *MusicLibrary) musicFilesDir() string {
	return paths.GetMusicFilesDir(l.DataRoot)
}

// AbsSongPath 把 Song.FilePath（"2026年第34周/05.mp3" 或暂存区裸文件名）
// 解析为绝对路径。调用方需自行确认文件存在。
func (l *MusicLibrary) AbsSongPath(rel string) string {
	name := filepath.FromSlash(strings.ReplaceAll(rel, "\\", "/"))
	if !filepath.IsLocal(name) {
		return ""
	}
	if strings.ContainsRune(name, filepath.Separator) {
		return filepath.Join(l.musicFilesDir(), name)
	}
	return filepath.Join(paths.GetSongStagingDir(l.DataRoot), name)
}

// ---------------------------------------------------------------------------
// 单首歌音频来源
// ---------------------------------------------------------------------------

// obtainSongSource 返回一份「只包含这首歌、带原始 ID3 标签」的独立 MP3 路径。
//
//   - FilePath 指向暂存区：暂存文件本身就是独立 MP3。
//   - FilePath 指向周文件夹且该编号是单曲：文件本身。
//   - FilePath 指向合并文件：按 Parts 信息抽出到临时文件。
//
// 返回的临时文件由调用方负责清理（cleanup 非 nil 时）。
func (l *MusicLibrary) obtainSongSource(id int64) (path string, cleanup func(), err error) {
	path, part, err := l.locateSongAudio(id)
	if err != nil || part == nil {
		return path, nil, err
	}
	tmpPath, err := l.extractPart(path, *part)
	if err != nil {
		return "", nil, err
	}
	return tmpPath, func() { _ = os.Remove(tmpPath) }, nil
}

func (l *MusicLibrary) locateSongAudio(id int64) (string, *models.MergedPart, error) {
	var song models.Song
	var found bool
	l.Repo.Read(func(d *repo.Data) {
		// 播音歌虽然不进周文件夹，但同样可能带音频（暂存区），按 ID 查找时一并覆盖
		for _, list := range [][]models.Song{d.DormSongs, d.BroadcastSongs} {
			for _, s := range list {
				if s.ID == id {
					song = s
					found = true
					return
				}
			}
			if found {
				return
			}
		}
	})
	if !found {
		return "", nil, fmt.Errorf("song %d not found", id)
	}
	if song.FilePath == "" {
		return "", nil, fmt.Errorf("song %d has no audio file", id)
	}

	abs := l.AbsSongPath(song.FilePath)
	if abs == "" || paths.IsKugouDataPath(l.DataRoot, abs) {
		return "", nil, fmt.Errorf("invalid audio path for song %d", id)
	}
	if !strings.ContainsAny(song.FilePath, `/\`) {
		// 暂存区：独立文件
		if _, statErr := os.Stat(abs); statErr != nil {
			return "", nil, fmt.Errorf("staging file missing for song %d", id)
		}
		return abs, nil, nil
	}

	// 周文件夹：查索引判断是单曲还是合并成员
	rel := filepath.ToSlash(song.FilePath)
	parts := strings.SplitN(rel, "/", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid song file path %q", song.FilePath)
	}
	weekName, fileName := parts[0], parts[1]
	var wf *models.WeekFile
	l.Repo.Read(func(d *repo.Data) {
		if m := d.Weeks[weekName]; m != nil {
			if f, ok := m[strings.TrimSuffix(fileName, ".mp3")]; ok {
				wf = &f
			}
		}
	})

	if wf != nil && wf.Kind == models.WeekFileMerged {
		for _, p := range wf.Parts {
			if p.SongID == id {
				return abs, &p, nil
			}
		}
		return "", nil, fmt.Errorf("song %d not found in merged file %s", id, song.FilePath)
	}

	// 单曲（或索引缺失）：文件本身
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", nil, fmt.Errorf("audio file missing for song %d: %s", id, abs)
	}
	return abs, nil, nil
}

// OpenSongAudio exposes an individual MP3 and holds the media read lock until
// the caller closes it. Merged tracks can serve byte ranges without extraction.
func (l *MusicLibrary) OpenSongAudio(id int64) (content io.ReadSeeker, size int64, modified time.Time, close func(), err error) {
	l.mediaMu.RLock()
	var file *os.File
	defer func() {
		if err != nil {
			if file != nil {
				_ = file.Close()
			}
			l.mediaMu.RUnlock()
		}
	}()
	path, part, err := l.locateSongAudio(id)
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}
	file, err = os.Open(path)
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, time.Time{}, nil, fmt.Errorf("not a regular audio file: %s", path)
	}
	close = func() {
		_ = file.Close()
		l.mediaMu.RUnlock()
	}
	if part == nil {
		return file, info.Size(), info.ModTime(), close, nil
	}
	section, err := mergedSongReader(file, info.Size(), *part)
	if err != nil {
		return nil, 0, time.Time{}, nil, err
	}
	return section, section.Size(), time.Time{}, close, nil
}

// extractPart 把合并文件里的一个成员还原成独立 MP3（原始标签 + 裸音频帧）。
func (l *MusicLibrary) extractPart(mergedPath string, part models.MergedPart) (string, error) {
	input, err := os.Open(mergedPath)
	if err != nil {
		return "", err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return "", err
	}
	section, err := mergedSongReader(input, info.Size(), part)
	if err != nil {
		return "", err
	}
	tmpDir := paths.GetTempDir(l.DataRoot)
	if err := paths.EnsureDir(tmpDir); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(tmpDir, "extract-*.mp3")
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := io.CopyN(tmp, section, section.Size()); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	ok = true
	tmpPath := tmp.Name()
	return tmpPath, nil
}

func mergedSongReader(audio io.ReaderAt, size int64, part models.MergedPart) (*io.SectionReader, error) {
	if part.Offset < 0 || part.Length < 0 || part.Offset > size || part.Length > size-part.Offset {
		return nil, fmt.Errorf("song %d audio range is outside the merged file", part.SongID)
	}
	tagReader := func(encoded string) *io.SectionReader {
		tag, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			tag = nil
		}
		return io.NewSectionReader(bytes.NewReader(tag), 0, int64(len(tag)))
	}
	header, footer := tagReader(part.ID3v2), tagReader(part.ID3v1)
	sections := songSections{header, io.NewSectionReader(audio, part.Offset, part.Length), footer}
	return io.NewSectionReader(sections, 0, header.Size()+part.Length+footer.Size()), nil
}

type songSections [3]*io.SectionReader

func (s songSections) ReadAt(p []byte, off int64) (int, error) {
	total := 0
	for _, section := range s {
		if off >= section.Size() {
			off -= section.Size()
			continue
		}
		n, err := section.ReadAt(p, off)
		total += n
		p = p[n:]
		if err != nil && (err != io.EOF || int64(n) != section.Size()-off) {
			return total, err
		}
		if len(p) == 0 {
			return total, nil
		}
		off = 0
	}
	return total, io.EOF
}

// ---------------------------------------------------------------------------
// 文件构建
// ---------------------------------------------------------------------------

// buildMerged 把同一时段的多首歌按顺序合并写到 outPath，并记录每个成员的
// 还原信息。与 converter.MergeMP3s 同一套拼接策略（只保留第一个文件的
// ID3v2、剥掉全部 ID3v1），但额外记录标签与字节区间。
func (l *MusicLibrary) buildMerged(songs []models.Song, outPath string) ([]models.MergedPart, error) {
	out, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("create merged file: %w", err)
	}
	fail := func(err error) ([]models.MergedPart, error) {
		out.Close()
		os.Remove(outPath)
		return nil, err
	}

	var parts []models.MergedPart
	var offset int64
	for i, s := range songs {
		srcPath, cleanup, srcErr := l.obtainSongSource(s.ID)
		if srcErr != nil {
			return fail(fmt.Errorf("obtain source for song %d: %w", s.ID, srcErr))
		}
		data, readErr := os.ReadFile(srcPath)
		if cleanup != nil {
			cleanup()
		}
		if readErr != nil {
			return fail(fmt.Errorf("read source for song %d: %w", s.ID, readErr))
		}
		v2, v1, frames := converter.SplitID3Tags(data)

		// 只保留第一个文件的 ID3v2 作为合并文件的头部标签
		if i == 0 && len(v2) > 0 {
			if _, wErr := out.Write(v2); wErr != nil {
				return fail(fmt.Errorf("write head tag: %w", wErr))
			}
			// 头部标签占据的长度也要计入偏移，part0 的裸帧并不是从 0 开始
			offset += int64(len(v2))
		}

		part := models.MergedPart{
			SongID: s.ID,
			Offset: offset,
			Length: int64(len(frames)),
		}
		if len(v2) > 0 {
			part.ID3v2 = base64.StdEncoding.EncodeToString(v2)
		}
		if len(v1) > 0 {
			part.ID3v1 = base64.StdEncoding.EncodeToString(v1)
		}
		if _, wErr := out.Write(frames); wErr != nil {
			return fail(fmt.Errorf("write frames: %w", wErr))
		}
		offset += int64(len(frames))
		parts = append(parts, part)
	}
	if err := out.Sync(); err != nil {
		return fail(fmt.Errorf("sync merged file: %w", err))
	}
	if err := out.Close(); err != nil {
		os.Remove(outPath)
		return nil, fmt.Errorf("close merged file: %w", err)
	}
	return parts, nil
}

// ---------------------------------------------------------------------------
// 同步
// ---------------------------------------------------------------------------

// SyncWeek 把 weekName 的周文件夹收敛到期望布局。
//
// 流程：
//  1. 计算期望布局，与 data.json 里记录的当前索引按「编号签名」比对；
//  2. 有变化的编号先构建成 .tmp-NN.mp3（构建期间旧文件仍是成员音频来源）；
//  3. 该周不再被引用的旧编号删除；
//  4. 临时文件改名就位；
//  5. 一次性写回 data.json：新索引 + 该周歌曲的 FilePath + 孤儿歌的暂存迁移。
func (l *MusicLibrary) CollectRetiredSongs(now time.Time) error {
	l.mediaMu.Lock()
	defer l.mediaMu.Unlock()
	return l.Repo.Collect(now)
}

func (l *MusicLibrary) RestoreRetiredSongs(id string, now time.Time) error {
	l.mediaMu.Lock()
	defer l.mediaMu.Unlock()
	return l.Repo.RestoreRecovery(id, now)
}

func (l *MusicLibrary) SyncWeek(weekName string) error {
	l.mediaMu.Lock()
	defer l.mediaMu.Unlock()
	return l.syncWeek(weekName)
}

func (l *MusicLibrary) syncWeek(weekName string) error {
	slots, err := l.computeLayout(weekName)
	if err != nil {
		return err
	}
	weekDir := filepath.Join(l.musicFilesDir(), weekName)
	if err := paths.EnsureDir(weekDir); err != nil {
		return fmt.Errorf("create week dir: %w", err)
	}

	var current map[string]models.WeekFile
	l.Repo.Read(func(d *repo.Data) {
		current = d.Weeks[weekName]
	})

	// 已存在的静音文件可以作为模板复制，避免每次同步都跑一遍 ffmpeg
	silentTemplate := ""
	for num, f := range current {
		if f.Kind == models.WeekFileSilent {
			p := filepath.Join(weekDir, num+".mp3")
			if _, statErr := os.Stat(p); statErr == nil {
				silentTemplate = p
				break
			}
		}
	}

	type result struct {
		num  string
		file models.WeekFile
		tmp  string // 构建产物（.tmp-NN.mp3）；keep 时为空
		keep bool
	}
	var results []result

	for _, slot := range slots {
		desired := desiredFile(slot)
		if cur, ok := current[slot.Num]; ok && cur.Signature() == desired.Signature() {
			p := filepath.Join(weekDir, slot.Num+".mp3")
			if _, statErr := os.Stat(p); statErr == nil {
				results = append(results, result{num: slot.Num, file: cur, keep: true})
				continue
			}
		}

		tmp := filepath.Join(weekDir, ".tmp-"+slot.Num+".mp3")
		os.Remove(tmp)
		switch desired.Kind {
		case models.WeekFileSilent:
			if silentTemplate != "" {
				if err := converter.CopyFile(silentTemplate, tmp); err != nil {
					return fmt.Errorf("copy silent template: %w", err)
				}
			} else if err := l.Converter.GenerateSilentPlaceholderMP3(tmp, converter.DefaultSilentPlaceholder); err != nil {
				return fmt.Errorf("generate silent placeholder: %w", err)
			}
		case models.WeekFileSong:
			srcPath, cleanup, srcErr := l.obtainSongSource(slot.Songs[0].ID)
			if srcErr != nil {
				return fmt.Errorf("obtain source: %w", srcErr)
			}
			err := converter.CopyFile(srcPath, tmp)
			if cleanup != nil {
				cleanup()
			}
			if err != nil {
				return fmt.Errorf("copy song file: %w", err)
			}
		case models.WeekFileMerged:
			parts, mErr := l.buildMerged(slot.Songs, tmp)
			if mErr != nil {
				return fmt.Errorf("build merged file: %w", mErr)
			}
			desired.Parts = parts
		}
		results = append(results, result{num: slot.Num, file: desired, tmp: tmp})
	}

	// 期望布局里出现过的编号
	desiredNums := make(map[string]bool, len(results))
	for _, r := range results {
		desiredNums[r.num] = true
	}

	// 孤儿歌：有音频、FilePath 指向本周文件夹、但不在期望布局里
	// （时段被删/未分配/日期改到了别的周）。在删除旧文件之前，把它们的音频
	// 抽到暂存区，FilePath 改指暂存文件。
	type relocation struct {
		songID  int64
		newPath string
	}
	desiredSongIDs := make(map[int64]bool)
	for _, slot := range slots {
		for _, s := range slot.Songs {
			desiredSongIDs[s.ID] = true
		}
	}
	var relocations []relocation
	var songs []models.Song
	l.Repo.Read(func(d *repo.Data) { songs = d.DormSongs })
	for _, s := range songs {
		if s.FilePath == "" || desiredSongIDs[s.ID] {
			continue
		}
		if !strings.HasPrefix(filepath.ToSlash(s.FilePath), weekName+"/") {
			continue
		}
		srcPath := l.AbsSongPath(s.FilePath)
		if _, statErr := os.Stat(srcPath); statErr != nil {
			continue // 文件已经不在了，无从抽取
		}
		// 单曲直接搬；合并成员抽出自己的部分
		standalone, cleanup, srcErr := l.obtainSongSource(s.ID)
		if srcErr != nil {
			continue
		}
		staging := paths.GetSongStagingDir(l.DataRoot)
		_ = paths.EnsureDir(staging)
		stageName := fmt.Sprintf("song-%d.mp3", s.ID)
		stagePath := filepath.Join(staging, stageName)
		cpErr := converter.CopyFile(standalone, stagePath)
		if cleanup != nil {
			cleanup()
		}
		if cpErr != nil {
			continue
		}
		relocations = append(relocations, relocation{songID: s.ID, newPath: stageName})
	}

	var moves []repo.FileMove
	for _, result := range results {
		if !result.keep && result.tmp != "" {
			from, _ := filepath.Rel(l.DataRoot, result.tmp)
			to, _ := filepath.Rel(l.DataRoot, filepath.Join(weekDir, result.num+".mp3"))
			moves = append(moves, repo.FileMove{From: from, To: to})
		}
	}

	// 写回索引与歌曲 FilePath
	err = l.Repo.WriteWithMoves(func(d *repo.Data) error {
		if d.Weeks == nil {
			d.Weeks = make(map[string]map[string]models.WeekFile)
		}
		newIndex := make(map[string]models.WeekFile, len(results))
		for _, r := range results {
			newIndex[r.num] = r.file
		}
		d.Weeks[weekName] = newIndex

		// 期望布局内歌曲指向新编号文件；孤儿歌指向暂存区
		newPathBySong := make(map[int64]string, len(slots)+len(relocations))
		for _, slot := range slots {
			for _, s := range slot.Songs {
				newPathBySong[s.ID] = weekName + "/" + slot.Num + ".mp3"
			}
		}
		for _, rel := range relocations {
			newPathBySong[rel.songID] = rel.newPath
		}
		for i := range d.DormSongs {
			if p, ok := newPathBySong[d.DormSongs[i].ID]; ok {
				d.DormSongs[i].FilePath = p
			}
		}
		return nil
	}, moves)
	if err != nil {
		return err
	}

	for number := range current {
		if !desiredNums[number] {
			_ = os.Remove(filepath.Join(weekDir, number+".mp3"))
		}
	}
	knownStaged := make(map[string]bool)
	for _, slot := range slots {
		for _, song := range slot.Songs {
			if !strings.ContainsAny(song.FilePath, `/\`) {
				knownStaged[song.FilePath] = true
			}
		}
	}
	l.cleanupStaging(knownStaged)
	return nil
}

// SyncAll 重排所有出现过的周（歌曲所在周 + 索引里已有的周）。
// 时段配置变化导致编号整体平移时调用。
func (l *MusicLibrary) SyncAll() error {
	weekSet := make(map[string]bool)
	l.Repo.Read(func(d *repo.Data) {
		for _, s := range d.DormSongs {
			if name, err := WeekNameForDate(s.Date); err == nil && s.FilePath != "" {
				weekSet[name] = true
			}
		}
		for name := range d.Weeks {
			weekSet[name] = true
		}
	})
	names := make([]string, 0, len(weekSet))
	for name := range weekSet {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := l.SyncWeek(name); err != nil {
			l.Logger("musiclibrary: sync week %s failed: %v", name, err)
			return err
		}
	}
	return nil
}

// SyncSongWeek 同步某首歌（按其当前日期）所在的周。
func (l *MusicLibrary) SyncSongWeek(song models.Song) error {
	name, err := WeekNameForDate(song.Date)
	if err != nil {
		return err
	}
	return l.SyncWeek(name)
}

// cleanupStaging 删除暂存区里已没有任何歌曲引用的文件。
func (l *MusicLibrary) cleanupStaging(knownStaged map[string]bool) {
	staging := paths.GetSongStagingDir(l.DataRoot)
	entries, err := os.ReadDir(staging)
	if err != nil {
		return
	}
	referenced := make(map[string]bool)
	l.Repo.Read(func(d *repo.Data) {
		// 播音歌虽然不进周文件夹编号，但音频也放在暂存区，同样要算作引用，
		// 否则 cleanupStaging 会把播音歌的源文件误删。
		for _, s := range d.DormSongs {
			if s.FilePath != "" && !strings.ContainsAny(s.FilePath, `/\`) {
				referenced[s.FilePath] = true
			}
		}
		for _, s := range d.BroadcastSongs {
			if s.FilePath != "" && !strings.ContainsAny(s.FilePath, `/\`) {
				referenced[s.FilePath] = true
			}
		}
	})
	for _, e := range entries {
		if e.Type().IsRegular() && !referenced[e.Name()] {
			if info, err := e.Info(); err == nil && (knownStaged[e.Name()] || time.Since(info.ModTime()) > 24*time.Hour) {
				_ = os.Remove(filepath.Join(staging, e.Name()))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 导出
// ---------------------------------------------------------------------------

// ExportWeekResult 是 ExportWeekToDesktop 的结果。
type ExportWeekResult struct {
	ConfirmNeeded bool
	ExistingFiles []string
	TargetDir     string
	FileCount     int
}

// ExportWeekToDesktop 先 SyncWeek 保证周文件夹为最终形态，再把整个周文件夹
// 复制到桌面。桌面已存在同名文件夹时，confirm=false 返回待覆盖文件列表。
func (l *MusicLibrary) ExportWeekToDesktop(weekName string, confirm bool) (*ExportWeekResult, error) {
	l.mediaMu.Lock()
	defer l.mediaMu.Unlock()
	if err := l.syncWeek(weekName); err != nil {
		return nil, fmt.Errorf("同步周文件夹失败: %w", err)
	}

	// 整周都是静音占位（没有任何真实歌曲）时不导出
	hasRealSong := false
	l.Repo.Read(func(d *repo.Data) {
		for _, f := range d.Weeks[weekName] {
			if f.Kind != models.WeekFileSilent {
				hasRealSong = true
			}
		}
	})
	if !hasRealSong {
		return nil, fmt.Errorf("该周没有歌曲")
	}

	desktop, err := DesktopDir()
	if err != nil {
		return nil, fmt.Errorf("无法定位桌面目录: %w", err)
	}
	srcDir := filepath.Join(l.musicFilesDir(), weekName)
	targetDir := filepath.Join(desktop, weekName)

	if _, statErr := os.Stat(targetDir); statErr == nil && !confirm {
		var existing []string
		entries, _ := os.ReadDir(targetDir)
		for _, e := range entries {
			existing = append(existing, e.Name())
		}
		return &ExportWeekResult{ConfirmNeeded: true, ExistingFiles: existing, TargetDir: targetDir}, nil
	}

	if err := paths.EnsureDir(targetDir); err != nil {
		return nil, fmt.Errorf("创建导出目录失败: %w", err)
	}
	count, err := copyDirFiles(srcDir, targetDir)
	if err != nil {
		return nil, err
	}
	return &ExportWeekResult{TargetDir: targetDir, FileCount: count}, nil
}

// copyDirFiles 把 srcDir 里的普通文件复制到 dstDir（不递归子目录）。
func copyDirFiles(srcDir, dstDir string) (int, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return 0, fmt.Errorf("读取周文件夹失败: %w", err)
	}
	count := 0
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(e.Name()), ".mp3") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		if err := converter.CopyFile(src, dst); err != nil {
			return count, fmt.Errorf("复制 %s 失败: %w", e.Name(), err)
		}
		count++
	}
	return count, nil
}

// DesktopDir 定位用户桌面目录（兼容 OneDrive 重定向）。
func DesktopDir() (string, error) {
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, "Desktop")
		if info, statErr := os.Stat(p); statErr == nil && info.IsDir() {
			return p, nil
		}
	}
	if one := os.Getenv("OneDrive"); one != "" {
		p := filepath.Join(one, "Desktop")
		if info, statErr := os.Stat(p); statErr == nil && info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("desktop directory not found")
}
