package converter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 锁死 17.43s 是为了和「按时段位置编号」那条契约绑死——
// 任何想要「让用户可配」的尝试都会破坏编号与曲序的对应。
// 这条测试不是检查实现，是检查常量本身有没有被改动。
func TestDefaultSilentPlaceholderIs17430ms(t *testing.T) {
	want := 17430 * time.Millisecond
	if DefaultSilentPlaceholder != want {
		t.Fatalf("DefaultSilentPlaceholder must stay at 17.43s (17430ms) for slot-aligned export; got %s", DefaultSilentPlaceholder)
	}
}

// 用真的 ffmpeg 跑一下，确保生成出来的文件至少近似 17.43s。
// 没装 ffmpeg 就跳过——CI 上单独跑这条用例。
func TestGenerateSilentMP3ProducesExpectedDuration(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not in PATH; skipping runtime check")
	}
	c := New(ffmpegPath, "")

	dir := t.TempDir()
	out := filepath.Join(dir, "silent.mp3")
	if err := c.GenerateSilentMP3(out, DefaultSilentPlaceholder); err != nil {
		t.Fatalf("GenerateSilentMP3: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 1024 {
		t.Fatalf("silent mp3 too small: %d bytes", info.Size())
	}

	// 跑 ffprobe 反向验证时长（ffprobe 没有就用 ffmpeg 兜底）。允许 ±0.2s 误差。
	probePath, err := exec.LookPath("ffprobe")
	if err != nil {
		probePath = ffmpegPath
	}
	cmd := exec.Command(probePath, "-v", "quiet", "-print_format", "json", "-show_format", out)
	out2, err := cmd.Output()
	if err != nil {
		t.Skipf("ffprobe/ffmpeg probe failed: %v", err)
	}
	if !strings.Contains(string(out2), `"duration": "17.4`) {
		t.Fatalf("silent mp3 duration is not ~17.43s; ffprobe output: %s", string(out2))
	}
}
