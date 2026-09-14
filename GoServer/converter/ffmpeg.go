package converter

import (
	"context"
	"fmt"
	"os"
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

	cmd := createCommand(ctx, probePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath,
	)
	cmd.WaitDelay = 5 * time.Second
	output, err := cmd.Output()
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe failed: %w", err)
	}

	return parseProbeOutput(output)
}

// ConvertToMP3 converts an audio file to MP3 format
func (c *FFMpegConverter) ConvertToMP3(inputPath, outputPath string) error {
	// Acquire semaphore slot with timeout to avoid permanent blocking
	select {
	case c.semaphore <- struct{}{}:
	case <-time.After(60 * time.Second):
		return fmt.Errorf("convert failed: concurrency semaphore timeout")
	}
	defer func() { <-c.semaphore }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := createCommand(ctx, c.ffmpegPath,
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	cmd.WaitDelay = 5 * time.Second
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up partial output on failure
		os.Remove(outputPath)
		return fmt.Errorf("convert failed: %w, output: %s", err, string(output))
	}
	return nil
}

// GenerateSilentMP3 用 ffmpeg anullsrc 生成指定时长的静音 MP3。
//
// duration 支持小数秒（time.Duration）。空时段导出的固定占位时长是
// DefaultSilentPlaceholder（17.43s），写死在这里就是为了和「按时段位置编号」
// 这条契约绑死——任何想要「让用户可配」的尝试都会破坏编号与曲序的对应。
func (c *FFMpegConverter) GenerateSilentMP3(outputPath string, duration time.Duration) error {
	return c.generateSilent(outputPath, duration, false)
}

// GenerateSilentPlaceholderMP3 生成周文件夹里空时段的静音占位文件。
// 与 GenerateSilentMP3 的区别：ID3 标签写入「空音频」，用户在播放器里浏览
// 周文件夹时能一眼认出哪些编号是占位、不是真歌。
func (c *FFMpegConverter) GenerateSilentPlaceholderMP3(outputPath string, duration time.Duration) error {
	return c.generateSilent(outputPath, duration, true)
}

func (c *FFMpegConverter) generateSilent(outputPath string, duration time.Duration, tagged bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ffmpeg 的 -t 接受浮点秒（"17.43"），用 time.Duration.Seconds() 直接给小数。
	dur := fmt.Sprintf("%.3f", duration.Seconds())
	args := []string{
		"-y",
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=mono",
		"-t", dur,
		"-acodec", "libmp3lame",
		"-q:a", "2",
	}
	if tagged {
		args = append(args,
			"-metadata", "title=空音频",
			"-metadata", "artist=空音频",
			"-metadata", "album=空音频",
		)
	}
	args = append(args, outputPath)
	cmd := createCommand(ctx, c.ffmpegPath, args...)
	cmd.WaitDelay = 5 * time.Second
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
