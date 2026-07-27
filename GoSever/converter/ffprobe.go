package converter

import (
	"encoding/json"
	"fmt"
)

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
		Title:  out.Format.Tags.Title,
		Artist: out.Format.Tags.Artist,
	}, nil
}
