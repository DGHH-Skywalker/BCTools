#!/bin/bash
# BroadcastTool 精简版 ffmpeg 构建脚本
# 只在 MSYS2 UCRT64 环境下测试过
set -e

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$ROOT_DIR/src/ffmpeg"
DIST_DIR="$ROOT_DIR/dist"

FFMPEG_TAG="n7.0.2"
FFMPEG_MIRROR_GITEE="https://gitee.com/mirrors/ffmpeg.git"
FFMPEG_MIRROR_PROXY="https://ghproxy.com/https://github.com/FFmpeg/FFmpeg.git"

mkdir -p "$SRC_DIR" "$DIST_DIR"

# --------------------------------------------------
# 1. 拉取 ffmpeg 源码（优先国内镜像，失败自动换代理）
# --------------------------------------------------
if [ ! -d "$SRC_DIR/.git" ]; then
    echo "[INFO] 正在克隆 ffmpeg 源码（优先 Gitee 镜像）..."
    if ! git clone --depth 1 --branch "$FFMPEG_TAG" "$FFMPEG_MIRROR_GITEE" "$SRC_DIR"; then
        echo "[WARN] Gitee 镜像失败，切换 GitHub 代理..."
        git clone --depth 1 --branch "$FFMPEG_TAG" "$FFMPEG_MIRROR_PROXY" "$SRC_DIR"
    fi
fi

cd "$SRC_DIR"

# --------------------------------------------------
# 2. configure：只保留 BroadcastTool 需要的音频能力
# --------------------------------------------------
# 注意：MP4/M4A 解复用器在 ffmpeg 里的名字就是带逗号的这一串
MOV_DEMUXER='mov,mp4,m4a,3gp,3g2,mj2'

PKG_CONFIG_PATH="/ucrt64/lib/pkgconfig:$PKG_CONFIG_PATH" \
./configure \
  --prefix="$DIST_DIR" \
  --target-os=mingw64 \
  --arch=x86_64 \
  --cc=x86_64-w64-mingw32-gcc \
  --cxx=x86_64-w64-mingw32-g++ \
  --pkg-config=pkg-config \
  --pkg-config-flags="--static" \
  --extra-cflags="-O3" \
  --extra-ldflags="-static" \
  --extra-libs="-lm" \
  --enable-static \
  --disable-shared \
  --enable-gpl \
  --enable-version3 \
  --enable-libmp3lame \
  --disable-everything \
  --enable-ffmpeg \
  --enable-ffprobe \
  --enable-encoder=libmp3lame \
  --enable-decoder='mp3,mp3float,flac,aac,aac_latm,alac,vorbis,opus,wmav1,wmav2,wmapro,wmavoice,wmalossless,pcm_s16le,pcm_s24le,pcm_s32le,pcm_u8,pcm_s8,pcm_f32le' \
  --enable-muxer=mp3 \
  --enable-demuxer="mp3,flac,wav,aac,${MOV_DEMUXER},ogg,asf" \
  --enable-filter='anullsrc,aformat,aresample' \
  --enable-protocol=file \
  --enable-parser='aac,aac_latm,ac3,flac,mpegaudio,vorbis,opus' \
  --disable-indevs \
  --disable-outdevs \
  --enable-indev=lavfi \
  --enable-swresample \
  --disable-doc \
  --disable-htmlpages \
  --disable-manpages \
  --disable-podpages \
  --disable-txtpages \
  --disable-iconv \
  --disable-zlib \
  --disable-bzlib \
  --disable-lzma \
  --disable-network \
  --disable-debug \
  --disable-ffplay \
  --disable-pthreads \
  --enable-w32threads \
  --disable-hwaccels \
  --disable-vaapi \
  --disable-vdpau \
  --disable-dxva2 \
  --disable-d3d11va

# --------------------------------------------------
# 3. 编译安装
# --------------------------------------------------
make -j"$(nproc)"
make install

# 默认 make install 会把可执行文件放在 $DIST_DIR/bin/，
# 为了独立分发，移动到 dist 根目录并清理开发文件。
mv -v "$DIST_DIR/bin/ffmpeg.exe" "$DIST_DIR/ffmpeg.exe"
mv -v "$DIST_DIR/bin/ffprobe.exe" "$DIST_DIR/ffprobe.exe"
rm -rf "$DIST_DIR/bin" "$DIST_DIR/include" "$DIST_DIR/lib" "$DIST_DIR/share"

# 复制许可证文件到 dist，方便分发
cp -v "$SRC_DIR/COPYING"* "$DIST_DIR/" 2>/dev/null || true

echo ""
echo "[OK] 构建完成，产物位于: $DIST_DIR"
ls -lh "$DIST_DIR/ffmpeg.exe" "$DIST_DIR/ffprobe.exe"
