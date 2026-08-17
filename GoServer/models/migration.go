package models

import "time"

// MigrationScope controls what gets included in the migration payload.
//
//   - "json"  : 歌名 + 时段 + 其它设置（一个 JSON 文件，无音频）
//   - "files" : json 全部 + 歌曲文件 + 快照 + 删除日志（自包含备份目录）
type MigrationScope string

const (
	MigrationScopeJSON  MigrationScope = "json"
	MigrationScopeFiles MigrationScope = "files"
)

// MigrationExportRequest is the body for POST /api/migration/export.
//
//   - TargetDir  用户选定的外层目录（最终子目录会建在它下面）
//   - Scope      json / files
//   - Confirm    true 表示已知目标子目录已存在，确认覆盖
type MigrationExportRequest struct {
	TargetDir string         `json:"targetDir"`
	Scope     MigrationScope `json:"scope"`
	Confirm   bool           `json:"confirm,omitempty"`
}

// MigrationExportResult is returned after a successful (or conflicted) export.
//
//   - BackupDir       实际子目录路径（无冲突时新建；有冲突且 Confirm=true 时已被覆盖）
//   - WrittenFiles    实际写入的文件相对路径列表（相对 BackupDir）
//   - SkippedFiles    源文件缺失但歌单仍引用的歌曲（仅 files 模式统计）
//   - SongCount       包含在 data.json 里的歌曲总数
type MigrationExportResult struct {
	BackupDir    string   `json:"backupDir"`
	WrittenFiles []string `json:"writtenFiles"`
	SkippedFiles []string `json:"skippedFiles"`
	SongCount    int      `json:"songCount"`
	// 单 JSON 模式走浏览器下载，不在响应里；文件模式才返回 BackupDir 给前端展示。
	JSONPayload *MigrationPayload `json:"jsonPayload,omitempty"`
}

// MigrationPreview is the pre-flight response that tells the frontend whether
// the chosen target directory already contains a same-named backup folder.
type MigrationPreview struct {
	BackupDir      string   `json:"backupDir"`
	Exists         bool     `json:"exists"`
	ExistingFiles  []string `json:"existingFiles,omitempty"`
	WillOverwrite  int      `json:"willOverwrite"`
	CollisionIndex int      `json:"collisionIndex"` // 实际算出的 x（1,2,3...）
}

// MigrationPayload is the JSON document placed inside both the single-JSON
// download and the file-mode backup directory. 未来若加新字段（v2 导入时识别），
// 走 SchemaVersion 区分。
type MigrationPayload struct {
	SchemaVersion  int              `json:"schemaVersion"`
	AppVersion     string           `json:"appVersion"`
	ExportedAt     time.Time        `json:"exportedAt"`
	DormSongs      []Song           `json:"dormSongs"`
	BroadcastSongs []Song           `json:"broadcastSongs"`
	Settings       Settings         `json:"settings"`
	DeletedLog     []DeletedSongLog `json:"deletedLog,omitempty"`
}

// MigrationImportRequest is the body for POST /api/migration/import.
//
//   - BackupDir 备份目录（必须含 data.json，可选含 songs/ 子目录）
//   - JSONData  直接传入整个 MigrationPayload（用于"从 JSON 文件"导入，
//               文件已经在浏览器里读好了——不走磁盘）
//   - Mode      "merge"（默认：保留现有数据，只追加新歌单 / 合并设置）
//               "replace"（清空现有 dorm / broadcast 歌单与设置后整体导入）
//
// BackupDir 与 JSONData 二选一：前者是目录导入（带音频），后者是单文件导入。
type MigrationImportRequest struct {
	BackupDir string          `json:"backupDir"`
	JSONData  *MigrationPayload `json:"jsonData,omitempty"`
	Mode      string          `json:"mode,omitempty"` // "merge" | "replace"；空 = merge
}

// MigrationImportResult 描述一次导入的统计。
type MigrationImportResult struct {
	InsertedDorm      int      `json:"insertedDorm"`
	InsertedBroadcast int      `json:"insertedBroadcast"`
	FilesCopied       int      `json:"filesCopied"`
	FilesMissing      []string `json:"filesMissing,omitempty"`
	TimeSlotsMerged   int      `json:"timeSlotsMerged"`
	SkippedEmpty      int      `json:"skippedEmpty"`
}
