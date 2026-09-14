package services

import (
	"io"
	"os"
	"path/filepath"

	"broadcast-tool/paths"
)

// 5.7.0 起数据目录从 %APPDATA%\BroadcastTool 迁到 <exe>\BctoolData。
// 迁移分两个阶段，都必须在服务可用之前完成：
//
//	阶段一 PrepareLegacyMigration（repo 初始化之前）：
//	    纯文件拷贝 data.json / deleted_songs_log.json / snapshots。
//	阶段二 FinishLegacyMigration（MusicLibrary 就绪之后）：
//	    把旧 songs 目录的 uuid 文件倒进暂存区，再 SyncAll 重建全部周文件夹。
//
// 旧 %APPDATA% 目录原样保留作备份，不做任何修改。
// 条件「BctoolData 里没有 data.json」保证迁移只发生一次；老用户重复安装、
// 新用户全新安装都不会触发。

// PrepareLegacyMigration 阶段一：拷贝数据文件。返回是否发生了迁移。
func PrepareLegacyMigration(dataRoot string, logger func(format string, args ...interface{})) bool {
	if logger == nil {
		logger = func(format string, args ...interface{}) {}
	}
	legacyDir, err := paths.GetLegacyAppDataDir()
	if err != nil {
		return false
	}
	legacyData := paths.GetDataFilePath(legacyDir)
	if _, err := os.Stat(legacyData); err != nil {
		return false // 没有旧数据
	}
	if _, err := os.Stat(paths.GetDataFilePath(dataRoot)); err == nil {
		return false // 新目录已有数据，不覆盖
	}

	logger("检测到旧版数据目录 %s，开始迁移到 %s", legacyDir, dataRoot)
	if err := paths.EnsureDir(dataRoot); err != nil {
		logger("迁移失败：创建数据目录失败: %v", err)
		return false
	}

	// data.json（存在性上面已确认）
	if err := copyFileIfExists(legacyData, paths.GetDataFilePath(dataRoot), logger); err != nil {
		logger("迁移失败：拷贝 data.json 失败: %v", err)
		return false
	}

	// 删除日志与快照（有则拷）
	copyFileIfExists(paths.GetDeletedSongsLogPath(legacyDir), paths.GetDeletedSongsLogPath(dataRoot), logger)
	copyDirIfExists(paths.GetSnapshotsDir(legacyDir), paths.GetSnapshotsDir(dataRoot), logger)

	return true
}

// FinishLegacyMigration 阶段二：重建周文件夹布局。
func FinishLegacyMigration(dataRoot string, lib *MusicLibrary, logger func(format string, args ...interface{})) {
	if logger == nil {
		logger = func(format string, args ...interface{}) {}
	}
	legacyDir, err := paths.GetLegacyAppDataDir()
	if err != nil {
		return
	}
	legacySongs := paths.GetLegacySongsDir(legacyDir)
	entries, err := os.ReadDir(legacySongs)
	if err != nil || len(entries) == 0 {
		logger("数据迁移完成")
		return
	}

	// 旧 uuid 文件全部倒进暂存区，SyncAll 会按歌曲日期归入各周文件夹
	staging := paths.GetSongStagingDir(dataRoot)
	if err := paths.EnsureDir(staging); err != nil {
		logger("迁移失败：创建暂存目录失败: %v", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(legacySongs, e.Name())
		dst := filepath.Join(staging, e.Name())
		if err := copyFileAtomic(src, dst); err != nil {
			logger("迁移警告：拷贝 %s 失败: %v", e.Name(), err)
		}
	}

	if err := lib.SyncAll(); err != nil {
		logger("迁移警告：重建周文件夹时出错: %v", err)
	}
	logger("数据迁移完成：歌曲文件已按周归档到 %s", paths.GetMusicFilesDir(dataRoot))
}

func copyFileIfExists(src, dst string, logger func(format string, args ...interface{})) error {
	if _, err := os.Stat(src); err != nil {
		return nil // 源不存在，跳过
	}
	if err := copyFileAtomic(src, dst); err != nil {
		return err
	}
	return nil
}

func copyDirIfExists(srcDir, dstDir string, logger func(format string, args ...interface{})) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return
	}
	if err := paths.EnsureDir(dstDir); err != nil {
		logger("迁移警告：创建目录 %s 失败: %v", dstDir, err)
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())
		if err := copyFileAtomic(src, dst); err != nil {
			logger("迁移警告：拷贝 %s 失败: %v", e.Name(), err)
		}
	}
}

// copyFileAtomic 先写临时文件再改名，避免半截文件破坏数据。
func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".migrating"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
