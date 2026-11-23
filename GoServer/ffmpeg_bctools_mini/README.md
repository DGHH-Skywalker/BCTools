# 精简 FFmpeg

该目录为 BCTools 构建 Windows 版 `ffmpeg.exe` 和 `ffprobe.exe`，只保留歌曲导入、信息读取、转码与静音生成所需能力。

## 保留能力

| 类别 | 内容 |
| --- | --- |
| 输出编码 | MP3（libmp3lame） |
| 输入解码 | MP3、FLAC、WAV、AAC/M4A、ALAC、Ogg Vorbis、Opus、WMA、PCM |
| 滤镜 | `anullsrc`、`aformat`、`aresample` |
| 输入设备 | `lavfi` |
| 协议 | `file` |
| 工具 | `ffmpeg`、`ffprobe` |

视频、字幕、网络协议、ffplay 和项目未使用的组件会被裁掉。MP3 合并不依赖这里的 concat 组件，相关处理在 BCTools 的 Go 代码中完成。

## 构建环境

在 Windows 的 MSYS2 UCRT64 中安装依赖：

```bash
pacman -Syu
pacman -S --needed base-devel \
  mingw-w64-ucrt-x86_64-toolchain \
  mingw-w64-ucrt-x86_64-nasm \
  curl zstd git make pkg-config
```

构建脚本会获取固定版本的 FFmpeg 源码，并校验配套 LAME 静态库。

## 构建

从仓库根目录进入本目录：

```bash
cd GoServer/ffmpeg_bctools_mini
bash build.sh
```

产物写入 `dist/`，至少包括：

```text
ffmpeg.exe
ffprobe.exe
COPYING.*
```

设置以下环境变量时，还会生成发布所需的完整对应源码包：

```bash
BCTOOLS_PACKAGE_FFMPEG_SOURCE=1 bash build.sh
```

源码包为 `dist/FFmpeg-Sources.tar.gz`，包含固定提交的 FFmpeg 源码、对应 LAME 源码包、构建脚本和许可证。

## 集成到 BCTools

仓库根目录的以下命令会复制二进制到 Go 嵌入目录：

```bash
npm run build:inject-ffmpeg
```

完整 `npm run build` 已包含该步骤。主程序运行时再将工具释放到 `BctoolData/bin`。

GitHub Actions 独立构建并缓存 FFmpeg，再将产物交给 Windows 应用构建。缓存键包含 `build.sh` 的摘要，构建脚本变化后会重新编译。

## 许可证

本目录的构建脚本和文档采用 MIT 许可证。编译后的 FFmpeg、ffprobe 因启用 `libmp3lame`、`--enable-gpl` 和 `--enable-version3`，生成物按 GPLv3 分发。

分发二进制时必须同时提供对应源码和许可证。Release 工作流会发布 `FFmpeg-Sources.tar.gz`，不要只复制 EXE 而省略源码包。
