package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"broadcast-tool/models"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"
	"broadcast-tool/validation"

	"github.com/xuri/excelize/v2"
)

// ImportService handles xlsx import logic for songs.
type ImportService struct {
	Songs     *songstore.SongStore
	Snapshots *snapshotstore.SnapshotStore
}

// NewImportService creates a new ImportService.
func NewImportService(songs *songstore.SongStore, snapshots *snapshotstore.SnapshotStore) *ImportService {
	return &ImportService{Songs: songs, Snapshots: snapshots}
}

// ImportXlsx parses an uploaded xlsx file and imports valid rows.
// The caller is responsible for validating the song type and saving the uploaded file to tempFile.
func (s *ImportService) ImportXlsx(tempFile, songType string) (*models.ImportResult, error) {
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("无法解析 xlsx 文件")
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx 文件没有工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败")
	}

	type vr struct{ date, title, remark string }
	var valid []vr
	var importErrors []models.ImportError
	var skipped int
	seen := make(map[string]bool)

	for i, row := range rows {
		if i == 0 {
			continue
		}
		rn := i + 1
		date, title, remark := "", "", ""
		if len(row) > 0 {
			date = strings.TrimSpace(row[0])
		}
		if len(row) > 2 {
			title = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			remark = strings.TrimSpace(row[3])
		}
		if date == "" && title == "" {
			skipped++
			continue
		}
		if !validation.IsValidDate(date) {
			importErrors = append(importErrors, models.ImportError{Row: rn, Message: "日期格式错误"})
			continue
		}
		if title == "" {
			importErrors = append(importErrors, models.ImportError{Row: rn, Message: "歌名不能为空"})
			continue
		}
		key := date + "|" + strings.ToLower(strings.TrimSpace(title))
		if seen[key] {
			importErrors = append(importErrors, models.ImportError{Row: rn, Message: "xlsx 内重复"})
			continue
		}
		if s.Songs.CheckDuplicate(songType, date, title) {
			importErrors = append(importErrors, models.ImportError{Row: rn, Message: "与已有数据重复"})
			continue
		}
		seen[key] = true
		valid = append(valid, vr{date, title, remark})
	}

	s.Snapshots.CreatePreImportSnapshot()
	now := time.Now().UTC()
	var ns []models.Song
	for _, v := range valid {
		ns = append(ns, models.Song{
			ID:        s.Songs.GetNextID(),
			Date:      v.date,
			Title:     v.title,
			Remark:    v.remark,
			CreatedAt: now,
		})
	}
	if err := s.Songs.ReplaceSongsByType(songType, ns); err != nil {
		return nil, fmt.Errorf("导入写入失败")
	}
	s.Snapshots.EnsureDailySnapshot()

	return &models.ImportResult{
		Inserted: len(valid),
		Skipped:  skipped,
		Errors:   importErrors,
	}, nil
}

// SaveUploadedFile copies the uploaded multipart file to a temp file and returns its path.
func (s *ImportService) SaveUploadedFile(src io.Reader, appDataDir string) (string, error) {
	tempFile := filepath.Join(appDataDir, "temp", fmt.Sprintf("import-%d.xlsx", time.Now().UnixNano()))
	if err := os.MkdirAll(filepath.Dir(tempFile), 0755); err != nil {
		return "", err
	}
	out, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		return "", err
	}
	return tempFile, nil
}
