# ffmpeg_bctools_mini

BroadcastTool 的精简版 ffmpeg/ffprobe，只保留项目用到的音频编解码能力。

## 目标

- 把完整版 ffmpeg（约 230 MB）裁剪到 10–25 MB 级别。
- 零外部 DLL 依赖，单个文件夹即可独立运行。
- 可直接随 [BroadcastTool](https://github.com/your-repo/BroadcastTool) 一起分发，也可以单独拆出来开源。

## 保留能力

| 类别 | 内容 |
|------|------|
| 输出编码 | MP3（libmp3lame） |
| 输入解码 | MP3、FLAC、WAV、AAC/M4A、ALAC、Ogg Vorbis、Opus、WMA、PCM |
| 滤镜 | `anullsrc`（静音生成）、`aformat`、`aresample` |
| 输入设备 | `lavfi` |
| 协议 | `file` |
| 工具 | `ffmpeg`、`ffprobe` |

其余视频、字幕、网络、ffplay 等全部移除。

## 构建环境

Windows + MSYS2 UCRT64：

```bash
pacman -Syu
pacman -S --needed base-devel mingw-w64-ucrt-x86_64-toolchain \
  mingw-w64-ucrt-x86_64-nasm \
  mingw-w64-ucrt-x86_64-lame \
  git make pkg-config
```

> 国内用户建议先把 MSYS2 源换成清华镜像，再安装依赖。

## 构建

```bash
cd ffmpeg_bctools_mini
bash build.sh
```

构建完成后产物在 `dist/`：

```text
dist/
├── ffmpeg.exe
├── ffprobe.exe
└── COPYING.*
```

## 集成到 BroadcastTool

根目录 `npm run build` 会从 `ffmpeg_bctools_mini/dist/` 复制 `ffmpeg.exe` 和 `ffprobe.exe` 到 Go 的嵌入目录，最终随 `bctools.exe` 一起分发。

也可以手动复制，或直接把 `ffmpeg_bctools_mini/dist` 加到 PATH。

## 许可证

- 本仓库中的构建脚本、文档采用 MIT 许可证（见 `LICENSE`）。
- 编译后的 `ffmpeg.exe` / `ffprobe.exe` 二进制遵循 ffmpeg 原许可证。由于启用了 `libmp3lame` 并打开了 `--enable-gpl --enable-version3`，生成物为 **GPLv3**。
- 分发时请务必附带 `dist/COPYING*` 许可证文件。
