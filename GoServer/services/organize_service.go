// Package services contains higher-level business logic that is shared by
// multiple handlers or too large to live directly in a handler.
package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
//
// 静音占位时长是 converter.DefaultSilentPlaceholder（17.43s 硬编码）——
// 不再读 settings，参见 silent.go 的注释。
type OrganizeService struct {
	AppDataDir         string
	Converter          *converter.FFMpegConverter
	ValidateSource     SourceValidator
	ValidateTargetName TargetNameValidator
	ValidateTargetDir  TargetDirValidator
}

// NewOrganizeService creates an OrganizeService with the required dependencies.
func NewOrganizeService(
	appDataDir string,
	c *converter.FFMpegConverter,
	srcVal SourceValidator,
	targetVal TargetNameValidator,
	dirVal TargetDirValidator,
) *OrganizeService {
	return &OrganizeService{
		AppDataDir:         appDataDir,
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
		// 校验全部源文件（多首歌合并时 Sources 里可能有多个）
		for _, src := range entry.EffectiveSources() {
			if _, err := s.ValidateSource(src); err != nil {
				return "", fmt.Errorf("source 非法: %s", src)
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

// Execute performs the organize operation (copy/move/merge/silent).
func (s *OrganizeService) Execute(req models.OrganizeRequest, targetDir string) models.OrganizeResponse {
	var resp models.OrganizeResponse
	for _, entry := range req.Entries {
		targetName, _ := s.ValidateTargetName(entry.TargetName)
		targetPath := filepath.Join(targetDir, targetName)

		sources := entry.EffectiveSources()

		// 该时段没歌：生成静音占位，保持曲序与歌单对齐。
		if len(sources) == 0 {
			if err := s.generateSilent(targetPath); err != nil {
				resp.Failed = append(resp.Failed, models.FailedItem{Source: "", Reason: "生成静音文件失败"})
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: "", Target: targetPath})
			}
			continue
		}

		// 先把所有源路径校验并解析为绝对路径，任一失败就整条跳过——
		// 合并到一半再失败会在 SD 卡上留下残缺文件。
		absSources := make([]string, 0, len(sources))
		failed := false
		for _, src := range sources {
			abs, err := s.ValidateSource(src)
			if err != nil {
				resp.Failed = append(resp.Failed, models.FailedItem{Source: src, Reason: "源文件不在允许范围内"})
				failed = true
				break
			}
			if _, err := os.Stat(abs); os.IsNotExist(err) {
				resp.Failed = append(resp.Failed, models.FailedItem{Source: src, Reason: "源文件不存在"})
				failed = true
				break
			}
			absSources = append(absSources, abs)
		}
		if failed {
			continue
		}

		// 同一时段多首歌：按顺序合并成一个 MP3。
		// 库里仍然分开存放，只有导出到 SD 卡时才合并。
		if len(absSources) > 1 {
			if err := converter.MergeMP3s(absSources, targetPath); err != nil {
				os.Remove(targetPath)
				resp.Failed = append(resp.Failed, models.FailedItem{
					Source: strings.Join(sources, " + "),
					Reason: "合并音频失败",
				})
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{
					Source: strings.Join(sources, " + "),
					Target: targetPath,
				})
			}
			// 合并模式下绝不删除源文件：它们仍要留在歌库里。
			continue
		}

		srcPath := absSources[0]
		entrySource := sources[0]

		if req.Mode == "copy" {
			if err := s.copyFile(srcPath, targetPath); err != nil {
				os.Remove(targetPath)
				resp.Failed = append(resp.Failed, models.FailedItem{Source: entrySource, Reason: "复制失败"})
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entrySource, Target: targetPath})
			}
		} else {
			if err := os.Rename(srcPath, targetPath); err != nil {
				if err := s.copyFile(srcPath, targetPath); err != nil {
					os.Remove(targetPath)
					resp.Failed = append(resp.Failed, models.FailedItem{Source: entrySource, Reason: "移动失败"})
				} else {
					os.Remove(srcPath)
					resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entrySource, Target: targetPath})
				}
			} else {
				resp.Successful = append(resp.Successful, models.OrganizeResult{Source: entrySource, Target: targetPath})
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
	// 固定 17.43s：与前端 HandleSilent 走同一个常量，SD 卡上同一序号文件
	// 的时长才不会因为走不同路径而对不上。
	return s.Converter.GenerateSilentMP3(targetPath, converter.DefaultSilentPlaceholder)
}

func (s *OrganizeService) copyFile(src, dst string) error {
	return converter.CopyFile(src, dst)
}
