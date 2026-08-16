package converter

import (
	"bytes"
	"encoding/binary"
)

// 解密后的音频格式嗅探。
//
// unlock-music 的 Decoder 接口只暴露 Validate() + io.Reader，不告诉调用方
// 解出来的是什么格式，而它内部的 sniff 包是 internal/ 的、无法导入。
// 所以这里按魔数自己判一遍，用途是：解密结果已经是 MP3 时跳过 LAME 重编码。
//
// 魔数表对齐 unlock-music.dev/cli/internal/sniff/audio.go。

// audioFormat 是嗅探出的容器/编码格式。
type audioFormat string

const (
	formatUnknown audioFormat = ""
	formatMP3     audioFormat = "mp3"
	formatFLAC    audioFormat = "flac"
	formatOGG     audioFormat = "ogg"
	formatWAV     audioFormat = "wav"
	formatWMA     audioFormat = "wma"
	formatMP4     audioFormat = "m4a"
)

var wmaMagic = []byte{
	0x30, 0x26, 0xb2, 0x75, 0x8e, 0x66, 0xcf, 0x11,
	0xa6, 0xd9, 0x00, 0xaa, 0x00, 0x62, 0xce, 0x6c,
}

// sniffAudioFormat 根据文件头判断音频格式。header 建议至少 16 字节。
func sniffAudioFormat(header []byte) audioFormat {
	switch {
	case bytes.HasPrefix(header, []byte("fLaC")):
		return formatFLAC
	case bytes.HasPrefix(header, []byte("OggS")):
		return formatOGG
	case bytes.HasPrefix(header, []byte("RIFF")):
		return formatWAV
	case bytes.HasPrefix(header, wmaMagic):
		return formatWMA
	case bytes.HasPrefix(header, []byte("ID3")):
		return formatMP3
	case isMPEGAudioFrame(header):
		// 没有 ID3v2 标签的裸 MP3：靠帧同步头识别。
		return formatMP3
	case hasMPEG4FtypBox(header):
		return formatMP4
	}
	return formatUnknown
}

// isMPEGAudioFrame 检查 MPEG audio 帧同步头（11 位全 1），并排掉保留值，
// 避免把任意 0xFF 开头的数据误判成 MP3。
func isMPEGAudioFrame(header []byte) bool {
	if len(header) < 4 {
		return false
	}
	if header[0] != 0xFF || header[1]&0xE0 != 0xE0 {
		return false
	}
	version := (header[1] >> 3) & 0x03
	if version == 0x01 { // reserved
		return false
	}
	layer := (header[1] >> 1) & 0x03
	if layer == 0x00 { // reserved
		return false
	}
	bitrate := (header[2] >> 4) & 0x0F
	if bitrate == 0x0F { // bad
		return false
	}
	sampleRate := (header[2] >> 2) & 0x03
	return sampleRate != 0x03 // reserved
}

// hasMPEG4FtypBox 检查 MPEG-4 容器的 ftyp box。
func hasMPEG4FtypBox(header []byte) bool {
	if len(header) < 8 {
		return false
	}
	size := binary.BigEndian.Uint32(header[0:4])
	if size < 8 || size > uint32(len(header)) {
		return false
	}
	return bytes.Equal(header[4:8], []byte("ftyp"))
}
