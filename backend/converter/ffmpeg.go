package converter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FFMpegConverter handles audio conversion operations
type FFMpegConverter struct {
	ffmpegPath  string
	ffprobePath string
	semaphore   chan struct{} // limits concurrent conversions
}

// New creates a new converter with the given ffmpeg/ffprobe paths
func New(ffmpegPath, ffprobePath string) *FFMpegConverter {
	return &FFMpegConverter{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		semaphore:   make(chan struct{}, 4), // max 4 concurrent conversions
	}
}

// ProbeResult holds the result of audio probing
type ProbeResult struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

// Probe extracts audio metadata using ffprobe (or ffmpeg as fallback)
func (c *FFMpegConverter) Probe(filePath string) (ProbeResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	probePath := c.ffprobePath
	// If ffprobe doesn't exist, use ffmpeg (which has compatible -show_format)
	if _, err := os.Stat(probePath); os.IsNotExist(err) {
		probePath = c.ffmpegPath
	}

	cmd := exec.CommandContext(ctx, probePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath,
	)
	output, err := cmd.Output()
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe failed: %w", err)
	}

	return parseProbeOutput(output)
}

// ConvertToMP3 converts an audio file to MP3 format
func (c *FFMpegConverter) ConvertToMP3(inputPath, outputPath string) error {
	// Acquire semaphore slot
	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.ffmpegPath,
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up partial output on failure
		os.Remove(outputPath)
		return fmt.Errorf("convert failed: %w, output: %s", err, string(output))
	}
	return nil
}

// GenerateSilentMP3 generates a silent MP3 file of specified duration
func (c *FFMpegConverter) GenerateSilentMP3(outputPath string, durationSec int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dur := fmt.Sprintf("%d", durationSec)
	cmd := exec.CommandContext(ctx, c.ffmpegPath,
		"-y",
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=mono",
		"-t", dur,
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("generate silent mp3 failed: %w, output: %s", err, string(output))
	}
	return nil
}

// SanitizeFileName removes path separators and dangerous characters from filenames
func SanitizeFileName(name string) string {
	// Remove any path separators
	result := filepath.Base(name)
	if result == "." || result == "/" || result == "\\" {
		return ""
	}
	return result
}
