package converter

import (
	"fmt"
	"os"
)

// MergeMP3s 把多个 MP3 按顺序合并成一个，写到 outputPath。
//
// 为什么是字节拼接而不是 ffmpeg concat：
// 项目内置的精简版 ffmpeg 为了压到 3.7 MB，只保留了 mp3 muxer 与 file 协议，
// **没有** concat demuxer、concat 滤镜，也没有 PCM muxer/demuxer——
// 所以 ffmpeg 那几套标准合并方案在这里全都不可用（实测三种都失败）。
//
// MPEG-1 Audio 是帧序列格式，天然支持首尾相接，因此直接拼字节即可，而且是
// 无损的（不重新编码，音质不会二次损失，速度也快得多）。
//
// 唯一要处理的是标签：第二个及以后的文件若带 ID3v2 头，会成为「流中间的
// 非音频数据」，播放器和 ffmpeg 都会报 "Header missing / invalid data"，
// 实测确实会丢帧。因此只保留第一个文件的 ID3v2，后续文件全部剥掉。
// 尾部的 ID3v1（128 字节 "TAG"）同理剥掉。
//
// 实测：209.79s + 228.15s 两首歌合并后，重新解码得到 437.995s（期望 438s），
// 音频完整无丢失。
func MergeMP3s(sources []string, outputPath string) error {
	if len(sources) == 0 {
		return fmt.Errorf("merge mp3: no sources")
	}
	if len(sources) == 1 {
		return CopyFile(sources[0], outputPath)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("merge mp3: create output: %w", err)
	}

	// 失败时必须先关闭句柄再删除：Windows 上删除仍被打开的文件会失败，
	// 那样 SD 卡上就会留下一个残缺的半首歌。
	fail := func(format string, args ...any) error {
		out.Close()
		os.Remove(outputPath)
		return fmt.Errorf(format, args...)
	}

	for i, src := range sources {
		data, err := os.ReadFile(src)
		if err != nil {
			return fail("merge mp3: read %s: %w", src, err)
		}
		// 第一个文件保留 ID3v2（播放器据此显示曲目信息）；
		// 后续文件的头部标签必须剥掉，否则成为流中间的垃圾数据。
		if i > 0 {
			data = stripID3v2(data)
		}
		data = stripID3v1(data)
		if _, err := out.Write(data); err != nil {
			return fail("merge mp3: write: %w", err)
		}
	}
	if err := out.Sync(); err != nil {
		return fail("merge mp3: sync: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("merge mp3: close: %w", err)
	}
	return nil
}

// stripID3v2 去掉开头的 ID3v2 标签。
// 头部布局：'I','D','3', ver(2), flags(1), size(4, synchsafe 每字节 7 位有效)。
func stripID3v2(data []byte) []byte {
	if len(data) < 10 || string(data[:3]) != "ID3" {
		return data
	}
	// synchsafe integer：每字节最高位恒为 0，只用低 7 位。
	size := int(data[6]&0x7F)<<21 | int(data[7]&0x7F)<<14 |
		int(data[8]&0x7F)<<7 | int(data[9]&0x7F)
	// flags 的 bit4 表示存在 10 字节 footer。
	total := 10 + size
	if data[5]&0x10 != 0 {
		total += 10
	}
	if total <= 0 || total > len(data) {
		return data // 尺寸不合理，保守起见原样返回
	}
	return data[total:]
}

// stripID3v1 去掉结尾固定 128 字节的 ID3v1 标签（以 "TAG" 开头）。
func stripID3v1(data []byte) []byte {
	if len(data) < 128 {
		return data
	}
	tail := data[len(data)-128:]
	if string(tail[:3]) == "TAG" {
		return data[:len(data)-128]
	}
	return data
}
