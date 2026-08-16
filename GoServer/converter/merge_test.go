package converter

import (
	"os"
	"path/filepath"
	"testing"
)

// 构造一个带 ID3v2 头的假 MP3：10 字节头 + size 字节标签体 + 音频负载。
func fakeMP3WithID3(payload string, tagBody int) []byte {
	out := []byte{'I', 'D', '3', 4, 0, 0}
	// synchsafe size，仅低 7 位有效
	out = append(out,
		byte((tagBody>>21)&0x7F),
		byte((tagBody>>14)&0x7F),
		byte((tagBody>>7)&0x7F),
		byte(tagBody&0x7F),
	)
	out = append(out, make([]byte, tagBody)...)
	return append(out, []byte(payload)...)
}

func TestStripID3v2(t *testing.T) {
	data := fakeMP3WithID3("AUDIO", 16)
	got := stripID3v2(data)
	if string(got) != "AUDIO" {
		t.Fatalf("expected bare audio payload, got %q", got)
	}
}

func TestStripID3v2LeavesUntaggedDataAlone(t *testing.T) {
	raw := []byte{0xFF, 0xFB, 0x90, 0x00, 1, 2, 3}
	if got := stripID3v2(raw); string(got) != string(raw) {
		t.Fatal("untagged data must pass through unchanged")
	}
}

// 标签尺寸不合理（超过文件长度）时必须原样返回，不能把整个文件切没了。
func TestStripID3v2IgnoresBogusSize(t *testing.T) {
	data := fakeMP3WithID3("A", 4)
	data[6], data[7], data[8], data[9] = 0x7F, 0x7F, 0x7F, 0x7F // 巨大的 size
	if got := stripID3v2(data); len(got) != len(data) {
		t.Fatalf("bogus tag size must be ignored, got len %d want %d", len(got), len(data))
	}
}

func TestStripID3v1(t *testing.T) {
	audio := []byte("AUDIOPAYLOAD")
	tag := make([]byte, 128)
	copy(tag, "TAG")
	got := stripID3v1(append(audio, tag...))
	if string(got) != "AUDIOPAYLOAD" {
		t.Fatalf("ID3v1 trailer not stripped, got %q", got)
	}
}

func TestStripID3v1LeavesNonTagTailAlone(t *testing.T) {
	data := make([]byte, 200)
	copy(data[len(data)-128:], "NOTATAG")
	if got := stripID3v1(data); len(got) != 200 {
		t.Fatal("a non-TAG tail must not be stripped")
	}
}

// 合并的核心契约：第一首的 ID3 保留（播放器靠它显示信息），
// 后续文件的 ID3 必须剥掉——留在流中间会让播放器报 "Header missing" 并丢帧。
func TestMergeMP3sStripsOnlyLaterTags(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp3")
	b := filepath.Join(dir, "b.mp3")
	os.WriteFile(a, fakeMP3WithID3("FIRST", 8), 0644)
	os.WriteFile(b, fakeMP3WithID3("SECOND", 8), 0644)

	out := filepath.Join(dir, "out.mp3")
	if err := MergeMP3s([]string{a, b}, out); err != nil {
		t.Fatalf("MergeMP3s: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	// 开头必须还是 ID3
	if string(got[:3]) != "ID3" {
		t.Fatal("first file's ID3 header must be preserved")
	}
	// 整个文件里只允许出现一次 ID3 头
	count := 0
	for i := 0; i+3 <= len(got); i++ {
		if string(got[i:i+3]) == "ID3" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 ID3 header, found %d (mid-stream tags corrupt playback)", count)
	}
	// 两段音频内容都要在
	if !contains(got, "FIRST") || !contains(got, "SECOND") {
		t.Fatal("merged output lost audio payload")
	}
}

// 顺序必须严格按传入顺序——SD 卡上的播放次序就是它。
func TestMergeMP3sPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp3")
	b := filepath.Join(dir, "b.mp3")
	os.WriteFile(a, []byte("AAAA"), 0644)
	os.WriteFile(b, []byte("BBBB"), 0644)

	out := filepath.Join(dir, "out.mp3")
	if err := MergeMP3s([]string{b, a}, out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	if string(got) != "BBBBAAAA" {
		t.Fatalf("expected BBBBAAAA (input order), got %q", got)
	}
}

func TestMergeMP3sSingleSourceIsPlainCopy(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp3")
	content := fakeMP3WithID3("ONLY", 8)
	os.WriteFile(a, content, 0644)

	out := filepath.Join(dir, "out.mp3")
	if err := MergeMP3s([]string{a}, out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(out)
	// 单文件不该被剥标签，应当逐字节相同
	if string(got) != string(content) {
		t.Fatal("single-source merge must copy the file verbatim, tags included")
	}
}

func TestMergeMP3sRejectsEmptyInput(t *testing.T) {
	if err := MergeMP3s(nil, filepath.Join(t.TempDir(), "o.mp3")); err == nil {
		t.Fatal("expected an error for zero sources")
	}
}

// 任一源文件缺失时必须整体失败，且不留下残缺输出——
// SD 卡上出现半首歌比直接报错更糟。
func TestMergeMP3sFailsCleanlyOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mp3")
	os.WriteFile(a, []byte("AAAA"), 0644)
	out := filepath.Join(dir, "out.mp3")

	if err := MergeMP3s([]string{a, filepath.Join(dir, "missing.mp3")}, out); err == nil {
		t.Fatal("expected an error when a source is missing")
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a failed merge must not leave a partial file behind")
	}
}

func contains(haystack []byte, needle string) bool {
	n := []byte(needle)
	for i := 0; i+len(n) <= len(haystack); i++ {
		match := true
		for j := range n {
			if haystack[i+j] != n[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
