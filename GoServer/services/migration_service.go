// Package services - 数据迁移：把歌单（json / json+音频）导出到用户指定的目录。
//
// 命名规则：`xxxx年xx月xx日广播站点歌工具数据备份x`，
// x 从 1 开始递增，避让同一天重复导出的命名冲突。
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"broadcast-tool/internal/repo"
	"broadcast-tool/internal/version"
	"broadcast-tool/models"
	"broadcast-tool/paths"
	"broadcast-tool/store/deletedlogstore"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/store/songstore"
)

// MigrationService 编排「读数据 → 序列化 → 写文件」全流程。
//
// 依赖只读：repo / songstore / settingstore / deletedlogstore。
// 不写回主数据；只往用户选定的外层目录里写新内容。
type MigrationService struct {
	AppDataDir string
	Repo       *repo.Repository
	Songs      *songstore.SongStore
	Settings   *settingstore.SettingsStore
	DeletedLog *deletedlogstore.DeletedLogStore
	Snapshots  string // snapshots 目录绝对路径（paths.GetSnapshotsDir）
}

func NewMigrationService(
	appDataDir string,
	r *repo.Repository,
	songs *songstore.SongStore,
	settings *settingstore.SettingsStore,
	deletedLog *deletedlogstore.DeletedLogStore,
) *MigrationService {
	return &MigrationService{
		AppDataDir: appDataDir,
		Repo:       r,
		Songs:      songs,
		Settings:   settings,
		DeletedLog: deletedLog,
		Snapshots:  paths.GetSnapshotsDir(appDataDir),
	}
}

// currentSchemaVersion 单一来源：往后加新字段时改这里。
const currentSchemaVersion = 1

// backupDirPrefix 决定子目录命名前缀；时间戳走东八区，避免跨时区同一天撞名。
//
//	完整格式：2026年08月17日广播站点歌工具数据备份1
const backupDirPrefixFmt = "2006年01月02日广播站点歌工具数据备份"

// buildBackupName returns "2026年08月17日广播站点歌工具数据备份N"，N 从 1 开始递增
// 直到 targetDir 里没有同名子目录为止。
//
// 即使用户今天已经导出过 5 次（1..5），第 6 次仍然会得到后缀 6。
func (s *MigrationService) buildBackupName(targetDir string) (string, int) {
	now := time.Now()
	basePrefix := now.Format(backupDirPrefixFmt)
	for n := 1; ; n++ {
		name := fmt.Sprintf("%s%d", basePrefix, n)
		if _, err := os.Stat(filepath.Join(targetDir, name)); err != nil {
			return name, n
		}
	}
}

// BuildPayload 读取所有要导出的数据，组装成 MigrationPayload。
//
//   - DormSongs / BroadcastSongs 直接走 SongStore 接口拿到（带 weekday 计算）。
//   - Settings 走 SettingsStore 拿 raw（含 AdminPasswordHash），导出后用户在新机器
//     恢复时连密码一起搬过去；落盘文件反正只在用户手里。
//   - DeletedLog：拿不到就当空切片，不让日志读失败阻塞整次迁移。
func (s *MigrationService) BuildPayload() (models.MigrationPayload, error) {
	dorm := s.Songs.GetDormSongs()
	broadcast := s.Songs.GetBroadcastSongs()
	settings := s.Settings.GetSettingsRaw()

	log, err := s.DeletedLog.ReadDeletedSongLog()
	if err != nil {
		// 单条错误不该阻塞整个迁移——日志只是辅助数据。
		log = []models.DeletedSongLog{}
	}

	// strip computed weekday（写入 JSON 跟 data.json 一致；导入时再算）。
	for i := range dorm {
		dorm[i].Weekday = ""
	}
	for i := range broadcast {
		broadcast[i].Weekday = ""
	}

	return models.MigrationPayload{
		SchemaVersion:  currentSchemaVersion,
		AppVersion:     version.Version,
		ExportedAt:     time.Now(),
		DormSongs:      dorm,
		BroadcastSongs: broadcast,
		Settings:       settings,
		DeletedLog:     log,
	}, nil
}

// Preview 检查目标外层目录里下一个候选子目录是否已存在，
// 给前端「是直接覆盖 / 还是换文件夹」的判断依据。
//
// 注意：返回的 BackupDir 总是「按当前时间算出来」的下一个候选；调用方拿到
// 后直接用同一个名字做 Export 即可。
func (s *MigrationService) Preview(targetDir string) (models.MigrationPreview, error) {
	abs, err := filepath.Abs(targetDir)
	if err != nil {
		return models.MigrationPreview{}, fmt.Errorf("无效路径")
	}
	clean := filepath.Clean(abs)
	if strings.Contains(clean, "..") {
		return models.MigrationPreview{}, fmt.Errorf("路径包含非法字符")
	}

	name, index := s.buildBackupName(clean)
	full := filepath.Join(clean, name)

	preview := models.MigrationPreview{
		BackupDir:      full,
		CollisionIndex: index,
	}

	info, err := os.Stat(full)
	if err != nil {
		if !os.IsNotExist(err) {
			return preview, fmt.Errorf("检查目标失败: %w", err)
		}
		// 不存在：无冲突。
		return preview, nil
	}
	if !info.IsDir() {
		return preview, fmt.Errorf("同名路径已存在但不是目录")
	}

	// 存在：列出里面的文件，给前端「要被覆盖的文件清单」预览。
	entries, err := os.ReadDir(full)
	if err != nil {
		return preview, fmt.Errorf("读取已存在目录失败: %w", err)
	}
	preview.Exists = true
	for _, e := range entries {
		preview.ExistingFiles = append(preview.ExistingFiles, e.Name())
		preview.WillOverwrite++
	}
	sort.Strings(preview.ExistingFiles)
	return preview, nil
}

// Export 是真正落盘的一步。流程：
//
//  1. 算子目录名（与 Preview 走相同规则）
//  2. 若已存在且 Confirm=false，返回 confirmNeeded + existingFiles（前端弹框）
//  3. 若 Confirm=true：把已存在目录整个删掉，再重建（这是「覆盖」语义）
//  4. 写 data.json / songs/* / snapshots/* / deleted_songs_log.json
//
// 错误会回退到「不删除原目录、不留半成品」的状态：所有写入先到一个临时子目录
// `<name>.tmp`，全部成功后才 rename 成正式名字。
func (s *MigrationService) Export(
	req models.MigrationExportRequest,
) (models.MigrationExportResult, error) {
	abs, err := filepath.Abs(req.TargetDir)
	if err != nil {
		return models.MigrationExportResult{}, fmt.Errorf("无效路径")
	}
	clean := filepath.Clean(abs)
	if strings.Contains(clean, "..") {
		return models.MigrationExportResult{}, fmt.Errorf("路径包含非法字符")
	}

	name, _ := s.buildBackupName(clean)
	finalDir := filepath.Join(clean, name)
	tmpDir := finalDir + ".tmp"

	// 冲突检查（与 Preview 行为对齐：判断的是最终子目录而不是外层目录）。
	if info, err := os.Stat(finalDir); err == nil {
		if !info.IsDir() {
			return models.MigrationExportResult{}, fmt.Errorf("同名路径已存在但不是目录")
		}
		if !req.Confirm {
			// 让 handler 层把它翻译成 confirmNeeded=true 的响应。
			return models.MigrationExportResult{
				BackupDir: finalDir,
			}, errConfirmNeeded
		}
		// Confirm=true：先删原目录（彻底「覆盖」语义：不要把旧子目录里的无关文件留下来）。
		if err := os.RemoveAll(finalDir); err != nil {
			return models.MigrationExportResult{}, fmt.Errorf("清理已存在目录失败: %w", err)
		}
	}

	// 清掉可能残留的临时目录（上次崩溃留下的）。
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return models.MigrationExportResult{}, fmt.Errorf("创建临时目录失败: %w", err)
	}

	payload, err := s.BuildPayload()
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return models.MigrationExportResult{}, fmt.Errorf("读取数据失败: %w", err)
	}

	written := []string{}
	skipped := []string{}

	// 1) 写 data.json（主数据）
	dataPath := filepath.Join(tmpDir, "data.json")
	if err := writeJSON(dataPath, payload); err != nil {
		_ = os.RemoveAll(tmpDir)
		return models.MigrationExportResult{}, fmt.Errorf("写入 data.json 失败: %w", err)
	}
	written = append(written, "data.json")

	// 2) files 模式：拷贝歌曲文件 + 快照 + 删除日志
	if req.Scope == models.MigrationScopeFiles {
		songsDir := paths.GetSongsDir(s.AppDataDir)
		songWritten, songSkipped, err := s.copySongs(payload, songsDir, filepath.Join(tmpDir, "songs"))
		if err != nil {
			_ = os.RemoveAll(tmpDir)
			return models.MigrationExportResult{}, err
		}
		written = append(written, songWritten...)
		skipped = append(skipped, songSkipped...)

		// 快照目录（可选；不存在就不写）
		if info, err := os.Stat(s.Snapshots); err == nil && info.IsDir() {
			snapWritten, err := copyDirContents(s.Snapshots, filepath.Join(tmpDir, "snapshots"))
			if err != nil {
				_ = os.RemoveAll(tmpDir)
				return models.MigrationExportResult{}, fmt.Errorf("复制快照失败: %w", err)
			}
			for _, p := range snapWritten {
				written = append(written, filepath.ToSlash(filepath.Join("snapshots", p)))
			}
		}

		// 删除日志（可选；不存在就不写）
		if entries, err := s.DeletedLog.ReadDeletedSongLog(); err == nil && len(entries) > 0 {
			logPath := filepath.Join(tmpDir, "deleted_songs_log.json")
			if err := writeJSON(logPath, entries); err != nil {
				_ = os.RemoveAll(tmpDir)
				return models.MigrationExportResult{}, fmt.Errorf("写入删除日志失败: %w", err)
			}
			written = append(written, "deleted_songs_log.json")
		}
	}

	// 全部成功 → rename 到最终位置（原子切换；中途崩溃只可能留下 tmpDir，可清理）
	if err := os.Rename(tmpDir, finalDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return models.MigrationExportResult{}, fmt.Errorf("重命名到最终目录失败: %w", err)
	}

	return models.MigrationExportResult{
		BackupDir:    finalDir,
		WrittenFiles: written,
		SkippedFiles: skipped,
		SongCount:    len(payload.DormSongs) + len(payload.BroadcastSongs),
	}, nil
}

// errConfirmNeeded 是 sentinel error：handler 层识别它转成 confirmNeeded 响应。
// 不走 fmt.Errorf 是为了让 handler 用 errors.Is 判别，避免被错误信息拼接污染日志。
type sentinelError string

func (e sentinelError) Error() string { return string(e) }

const errConfirmNeeded sentinelError = "migration target exists; confirm required"

// IsConfirmNeeded reports whether the error means "target exists, ask frontend".
func IsConfirmNeeded(err error) bool {
	return err != nil && strings.Contains(err.Error(), errConfirmNeeded.Error())
}

// BuildPayloadJSON 是 json 模式专用的便捷方法：跳过文件写入，直接返回序列化字节。
func (s *MigrationService) BuildPayloadJSON() ([]byte, error) {
	payload, err := s.BuildPayload()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(payload, "", "  ")
}

// Import 从指定备份目录或 JSON 数据还原。两种入口：
//
//  1. 目录导入：req.BackupDir 指向 <backupDir>/data.json，可选含 songs/
//  2. JSON 导入：req.JSONData 直接是 MigrationPayload（浏览器读好整文件传进来）
//
// 流程：
//  1. 拿到 payload，校验 schemaVersion
//  2. 目录模式：把 songs/*.mp3 复制到 %APPDATA%\BroadcastTool\songs\
//  3. 给每首歌清空 ID（让 SongStore 重新分配），写入主数据
//  4. 合并 settings：TimeSlots 按 ID 去重，BroadcastColumnMap 缺哪补哪
//  5. mode="replace"：先清空本机 dorm / broadcast / settings.TimeSlots 再导入
//
// 错误会保留原状态：所有写入先在内存里组装，最后一次性 repo.Write 提交。
func (s *MigrationService) Import(req models.MigrationImportRequest) (models.MigrationImportResult, error) {
	result := models.MigrationImportResult{FilesMissing: []string{}}

	// 1) 取 payload：目录 vs JSON
	var payload models.MigrationPayload
	if req.JSONData != nil {
		payload = *req.JSONData
	} else {
		if req.BackupDir == "" {
			return result, fmt.Errorf("backupDir 与 jsonData 至少填一个")
		}
		abs, err := filepath.Abs(req.BackupDir)
		if err != nil {
			return result, fmt.Errorf("无效路径")
		}
		clean := filepath.Clean(abs)
		if strings.Contains(clean, "..") {
			return result, fmt.Errorf("路径包含非法字符")
		}
		dataPath := filepath.Join(clean, "data.json")
		raw, err := os.ReadFile(dataPath)
		if err != nil {
			return result, fmt.Errorf("读取 data.json 失败: %w", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return result, fmt.Errorf("data.json 解析失败: %w", err)
		}
		// 目录模式才复制音频
		songsDir := paths.GetSongsDir(s.AppDataDir)
		if err := paths.EnsureDir(songsDir); err != nil {
			return result, fmt.Errorf("创建 songs 目录失败: %w", err)
		}
		backupSongsDir := filepath.Join(clean, "songs")
		copied, missing := s.copyImportedSongs(payload, backupSongsDir, songsDir)
		result.FilesCopied = copied
		result.FilesMissing = missing
	}

	if payload.SchemaVersion != 0 && payload.SchemaVersion != currentSchemaVersion {
		return result, fmt.Errorf(
			"备份 schemaVersion=%d 与当前 %d 不匹配，无法导入",
			payload.SchemaVersion, currentSchemaVersion,
		)
	}
	payload.SchemaVersion = currentSchemaVersion

	// 2) strip computed weekday 与旧 ID（让 SongStore 重新分配）
	for i := range payload.DormSongs {
		payload.DormSongs[i].Weekday = ""
		payload.DormSongs[i].ID = 0
	}
	for i := range payload.BroadcastSongs {
		payload.BroadcastSongs[i].Weekday = ""
		payload.BroadcastSongs[i].ID = 0
	}

	// 3) 写回主数据
	mode := req.Mode
	if mode == "" {
		mode = "merge"
	}
	err := s.Repo.Write(func(d *repo.Data) error {
		if mode == "replace" {
			d.DormSongs = []models.Song{}
			d.BroadcastSongs = []models.Song{}
		}
		for _, sg := range payload.DormSongs {
			if sg.Title == "" && sg.FilePath == "" {
				result.SkippedEmpty++
				continue
			}
			d.DormSongs = append(d.DormSongs, sg)
			result.InsertedDorm++
		}
		for _, sg := range payload.BroadcastSongs {
			if sg.Title == "" && sg.FilePath == "" {
				result.SkippedEmpty++
				continue
			}
			d.BroadcastSongs = append(d.BroadcastSongs, sg)
			result.InsertedBroadcast++
		}
		if mode == "replace" {
			d.Settings.TimeSlots = payload.Settings.TimeSlots
			d.Settings.BroadcastColumnMap = payload.Settings.BroadcastColumnMap
		} else {
			existingIDs := map[string]struct{}{}
			for _, ts := range d.Settings.TimeSlots {
				existingIDs[ts.ID] = struct{}{}
			}
			for _, ts := range payload.Settings.TimeSlots {
				if _, dup := existingIDs[ts.ID]; dup {
					continue
				}
				d.Settings.TimeSlots = append(d.Settings.TimeSlots, ts)
				existingIDs[ts.ID] = struct{}{}
				result.TimeSlotsMerged++
			}
			if d.Settings.BroadcastColumnMap == nil {
				d.Settings.BroadcastColumnMap = map[string]string{}
			}
			for k, v := range payload.Settings.BroadcastColumnMap {
				if _, ok := d.Settings.BroadcastColumnMap[k]; !ok {
					d.Settings.BroadcastColumnMap[k] = v
				}
			}
		}
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("写入主数据失败: %w", err)
	}
	return result, nil
}

// copyImportedSongs 把 backupSongsDir 下被 payload 引用的所有歌曲文件复制到
// dstSongsDir。已存在则覆盖（用户已经在 modal 里点过「覆盖」）。
func (s *MigrationService) copyImportedSongs(
	payload models.MigrationPayload,
	backupSongsDir, dstSongsDir string,
) (copied int, missing []string) {
	if info, err := os.Stat(backupSongsDir); err != nil || !info.IsDir() {
		// 备份目录没有 songs/：纯 JSON 导出场景，跳过
		return 0, nil
	}
	seen := map[string]struct{}{}
	copyOne := func(filePath string) {
		base := filepath.Base(filePath)
		if base == "" || base == "." || base == ".." {
			return
		}
		if _, dup := seen[base]; dup {
			return
		}
		seen[base] = struct{}{}

		src := filepath.Join(backupSongsDir, base)
		if _, err := os.Stat(src); err != nil {
			missing = append(missing, base)
			return
		}
		dst := filepath.Join(dstSongsDir, base)
		if err := copyFile(src, dst); err != nil {
			missing = append(missing, base)
			return
		}
		copied++
	}
	for _, sg := range payload.DormSongs {
		copyOne(sg.FilePath)
	}
	for _, sg := range payload.BroadcastSongs {
		copyOne(sg.FilePath)
	}
	sort.Strings(missing)
	return copied, missing
}

// copySongs 把 payload 里所有被引用到的歌曲文件复制到 dstSongsDir。
//
//   - 只复制「歌单里引用的」文件，不复制 %APPDATA% 整个 songs/ 目录（避免搬运
//     已被删但文件还残留的死文件）。
//   - 源文件缺失就记到 skipped（用户在新机器恢复时知道这首歌丢了）。
//   - 返回的 written/skipped 都是相对 dstSongsDir 的路径，方便前端展示。
func (s *MigrationService) copySongs(
	payload models.MigrationPayload,
	srcSongsDir, dstSongsDir string,
) (written, skipped []string, err error) {
	if err := os.MkdirAll(dstSongsDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建 songs 目录失败: %w", err)
	}

	seen := map[string]struct{}{}
	copyOne := func(filePath string) {
		if filePath == "" {
			return
		}
		base := filepath.Base(filePath)
		if _, dup := seen[base]; dup {
			return
		}
		seen[base] = struct{}{}

		src := filepath.Join(srcSongsDir, base)
		dst := filepath.Join(dstSongsDir, base)
		if _, statErr := os.Stat(src); statErr != nil {
			skipped = append(skipped, base)
			return
		}
		if copyErr := copyFile(src, dst); copyErr != nil {
			skipped = append(skipped, base)
			return
		}
		written = append(written, base)
	}

	for _, s := range payload.DormSongs {
		copyOne(s.FilePath)
	}
	for _, s := range payload.BroadcastSongs {
		copyOne(s.FilePath)
	}
	sort.Strings(written)
	sort.Strings(skipped)
	return written, skipped, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// copyDirContents 把 srcDir 下所有文件浅拷贝到 dstDir（不递归；snapshots 没有子目录需求）。
func copyDirContents(srcDir, dstDir string) ([]string, error) {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, err
	}
	var written []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if err := copyFile(filepath.Join(srcDir, name), filepath.Join(dstDir, name)); err != nil {
			return written, err
		}
		written = append(written, name)
	}
	sort.Strings(written)
	return written, nil
}
