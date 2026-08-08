// Package services contains higher-level business logic that is shared by
// multiple handlers or too large to live directly in a handler.
package services

import (
	"fmt"
	"os"
	"path/filepath"

	"broadcast-tool/converter"
	"broadcast-tool/models"
	"broadcast-tool/paths"
)

// SourceValidator validates a source file path and returns the absolute path.
type SourceValidator func(src string) (string, error)

// TargetNameValidator validates and sanitizes a target file name.
type TargetNameValidator func(name string) (string, error)

// TargetDirValidator validates and returns the absolute target directory.
type TargetDirValidator func(dir string) (string, error)

// OrganizeService orchestrates file copy/move/silent generation for the
// "换卡工具" (organize) endpoint.
type OrganizeService struct {
	AppDataDir         string
	SettingsStore      interface{ GetSilentDuration() int }
	Converter          *converter.FFMpegConverter
	ValidateSource     SourceValidator
	ValidateTargetName TargetNameValidator
	ValidateTargetDir  TargetDirValidator
}

// NewOrganizeService creates an OrganizeService with the required dependencies.
func NewOrganizeService(
	appDataDir string,
	settings interface{ GetSilentDuration() int },
	c *converter.FFMpegConverter,
	srcVal SourceValidator,
	targetVal TargetNameValidator,
	dirVal TargetDirValidator,
) *OrganizeService {
	return &OrganizeService{
		AppDataDir:         appDataDir,
		SettingsStore:      settings,
		Converter:          c,
		ValidateSource:     srcVal,
		ValidateTargetName: targetVal,
		ValidateTargetDir:  dirVal,
	}
}

// PreValidate checks all entries before performing the actual operation.
// It returns the sanitized target directory and any validation error.
func (s *OrganizeService) PreValidate(req models.OrganizeRequest) (string, error) {
	targetDir, err := s.ValidateTargetDir(req.TargetDir)
	if err != nil {
		return "", err
	}
	for _, entry := range req.Entries {
		if _, err := s.ValidateTargetName(entry.TargetName); err != nil {
			return "", fmt.Errorf("targetName 非法: %s", entry.TargetName)
		}
		if entry.Source != "" {
			if _, err := s.ValidateSource(entry.Source); err != nil {
				return "", fmt.Errorf("source 非法: %s", entry.Source)
			}
		}
	}
	return targetDir, nil
}

// ListExistingFiles returns the list of target files that already exist.
func (s *OrganizeService) ListExistingFiles(entries []models.OrganizeEntry, targetDir string) []string {
	var existing []string
	for _, entry := range entries {
		targetName, _ := s.ValidateTargetName(entry.TargetName)
		targetPath := filepath.Join(targetDir, targetName)
		if _, err := os.Stat(targetPath); err == nil {
			existing = append(existing, targetName)
		}
	}
	return existing
}

// Execute performs the organize operation (copy/move/silent).
func (s *OrganizeService) Execute(req models.OrganizeRequest, targetDir string) models.OrganizeResponse {
	var resp models.OrganizeResponse
	for _, entry := range req.Entries {
		targetName, _ := s.ValidateTargetName(entry.TargetName)
		targetPath := filepath.Join(targetDir, targetName)

		if entry.Source == "" {
			if err := s.generateSilent(targetPath); err != nil {
				resp.Failed = append(resp.Failed, models.FailedItem{Source: "", Reason: "生成静音文件失败"})
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: "", Target: targetPath})
			}
			continue
		}

		srcPath, err := s.ValidateSource(entry.Source)
		if err != nil {
			resp.Failed = append(resp.Failed, models.FailedItem{Source: entry.Source, Reason: "源文件不在允许范围内"})
			continue
		}
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			resp.Failed = append(resp.Failed, models.FailedItem{Source: entry.Source, Reason: "源文件不存在"})
			continue
		}

		if req.Mode == "copy" {
			if err := s.copyFile(srcPath, targetPath); err != nil {
				os.Remove(targetPath)
				resp.Failed = append(resp.Failed, models.FailedItem{Source: entry.Source, Reason: "复制失败"})
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entry.Source, Target: targetPath})
			}
		} else {
			if err := os.Rename(srcPath, targetPath); err != nil {
				if err := s.copyFile(srcPath, targetPath); err != nil {
					os.Remove(targetPath)
					resp.Failed = append(resp.Failed, models.FailedItem{Source: entry.Source, Reason: "移动失败"})
				} else {
					os.Remove(srcPath)
					resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entry.Source, Target: targetPath})
				}
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entry.Source, Target: targetPath})
			}
		}
	}
	return resp
}

func (s *OrganizeService) generateSilent(targetPath string) error {
	tempDir := paths.GetTempDir(s.AppDataDir)
	if err := paths.EnsureDir(tempDir); err != nil {
		return err
	}
	dur := s.SettingsStore.GetSilentDuration()
	return s.Converter.GenerateSilentMP3(targetPath, dur)
}

func (s *OrganizeService) copyFile(src, dst string) error {
	return converter.CopyFile(src, dst)
}
