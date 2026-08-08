package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// CleanMetadata removes common junk placeholders and normalizes whitespace.
func CleanMetadata(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	replacements := []string{
		" - 未知艺术家",
		" - Unknown Artist",
		" - 群星",
		" - Various Artists",
	}
	for _, r := range replacements {
		s = strings.ReplaceAll(s, r, "")
	}
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// ffprobeOutput represents the JSON output from ffprobe -show_format
type ffprobeOutput struct {
	Format struct {
		Filename string `json:"filename"`
		Tags     struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
		} `json:"tags"`
	} `json:"format"`
}

// parseProbeOutput parses the JSON output from ffprobe/ffmpeg
func parseProbeOutput(data []byte) (ProbeResult, error) {
	var out ffprobeOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return ProbeResult{}, fmt.Errorf("parse probe output: %w", err)
	}
	return ProbeResult{
		Title:  CleanMetadata(out.Format.Tags.Title),
		Artist: CleanMetadata(out.Format.Tags.Artist),
	}, nil
}
