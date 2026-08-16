package converter

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSniffAudioFormat(t *testing.T) {
	mp4 := make([]byte, 16)
	binary.BigEndian.PutUint32(mp4[0:4], 16)
	copy(mp4[4:8], "ftyp")
	copy(mp4[8:12], "M4A ")

	cases := []struct {
		name   string
		header []byte
		want   audioFormat
	}{
		{"ID3v2 mp3", []byte("ID3\x04\x00\x00\x00\x00\x00\x00extra!!"), formatMP3},
		// 无 ID3 标签的裸 MP3 必须也认出来，否则会被白白重编码一遍。
		{"bare mp3 frame", []byte{0xFF, 0xFB, 0x90, 0x00, 0, 0, 0, 0, 0, 0, 0, 0}, formatMP3},
		{"flac", []byte("fLaC\x00\x00\x00\x22aaaaaaaa"), formatFLAC},
		{"ogg", []byte("OggS\x00\x02\x00\x00aaaaaaaa"), formatOGG},
		{"wav", []byte("RIFF\x24\x08\x00\x00WAVEfmt "), formatWAV},
		{"wma", append(append([]byte{}, wmaMagic...), 0, 0), formatWMA},
		{"mp4", mp4, formatMP4},
		{"garbage", []byte("not audio at all!"), formatUnknown},
		{"empty", []byte{}, formatUnknown},
		{"too short", []byte{0xFF}, formatUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sniffAudioFormat(c.header); got != c.want {
				t.Fatalf("sniffAudioFormat(%s) = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// 0xFF 开头但各字段是保留值的数据不能被当成 MP3——否则会把损坏/非音频内容
// 直接搬运成 .mp3，用户拿到一个放不出声的文件。
func TestSniffRejectsInvalidMPEGFrames(t *testing.T) {
	cases := []struct {
		name   string
		header []byte
	}{
		{"reserved version", []byte{0xFF, 0xE8 | 0x08, 0x90, 0x00}},
		{"reserved layer", []byte{0xFF, 0xFF & 0xF9, 0x90, 0x00}},
		{"bad bitrate", []byte{0xFF, 0xFB, 0xF0, 0x00}},
		{"reserved samplerate", []byte{0xFF, 0xFB, 0x9C, 0x00}},
		{"no sync bits", []byte{0xFF, 0x00, 0x90, 0x00}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sniffAudioFormat(c.header); got == formatMP3 {
				t.Fatalf("%s was wrongly sniffed as mp3", c.name)
			}
		})
	}
}

func TestSniffFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(p, []byte("ID3\x04\x00\x00\x00\x00\x00\x00payload"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := sniffFile(p)
	if err != nil {
		t.Fatalf("sniffFile: %v", err)
	}
	if got != formatMP3 {
		t.Fatalf("got %q, want mp3", got)
	}
}

func TestSniffFileMissing(t *testing.T) {
	if _, err := sniffFile(filepath.Join(t.TempDir(), "nope.bin")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

// toMP3 对已是 MP3 的输入必须直接搬运，不调用 ffmpeg。
// 这里把 ffmpegPath 设成一个不存在的程序：如果实现回退到转码，必然报错，
// 从而证明「没走 ffmpeg」这件事，而不是只看结果文件存在。
func TestToMP3SkipsReencodeForMP3(t *testing.T) {
	dir := t.TempDir()
	c := New(filepath.Join(dir, "definitely-not-ffmpeg.exe"), "")

	src := filepath.Join(dir, "decrypted.tmp")
	content := []byte("ID3\x04\x00\x00\x00\x00\x00\x00audio-bytes-here")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.mp3")

	if err := c.toMP3(src, out); err != nil {
		t.Fatalf("toMP3 on an mp3 input must not invoke ffmpeg, got: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	// 字节必须原样保留：重编码会改变内容。
	if !bytes.Equal(got, content) {
		t.Fatalf("output bytes differ from input; audio was re-encoded rather than copied")
	}
}

// 非 MP3 的解密结果必须真的走转码。ffmpeg 路径无效时应当报错，
// 而不是把 flac 原样改名成 .mp3 蒙过去。
func TestToMP3ConvertsNonMP3(t *testing.T) {
	dir := t.TempDir()
	c := New(filepath.Join(dir, "definitely-not-ffmpeg.exe"), "")

	src := filepath.Join(dir, "decrypted.tmp")
	if err := os.WriteFile(src, []byte("fLaC\x00\x00\x00\x22flac-payload"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.mp3")

	if err := c.toMP3(src, out); err == nil {
		t.Fatal("expected an error: a flac input must go through ffmpeg, not be copied as-is")
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("a failed conversion must not leave an output file behind")
	}
}
