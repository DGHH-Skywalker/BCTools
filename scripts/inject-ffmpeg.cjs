#!/usr/bin/env node
// 把 GoServer/ffmpeg_bctools_mini/dist/{ffmpeg,ffprobe}.exe 复制到
// GoServer/internal/binembed/bin/，让 go:embed 打进单文件 exe。
//
// 重要：只允许使用项目内置精简版 ffmpeg/ffprobe，不允许回退到外部完整版。
const fs = require("fs")
const path = require("path")

const ROOT = path.resolve(__dirname, "..")
const SRC = path.join(ROOT, "GoServer", "ffmpeg_bctools_mini", "dist")
const DST = path.join(ROOT, "GoServer", "internal", "binembed", "bin")

function log(m) { process.stdout.write(`[ffmpeg] ${m}\n`) }
function fail(m) { process.stderr.write(`[ffmpeg] ${m}\n`); process.exit(1) }

if (!fs.existsSync(SRC)) {
  fail(`未找到精简版 ffmpeg 目录: ${SRC}\n` +
       `请在 GoServer/ffmpeg_bctools_mini 下执行 build.sh 后再构建，\n` +
       `或从已有构建拷贝 dist/ffmpeg.exe / dist/ffprobe.exe 到该目录。`)
}

fs.mkdirSync(DST, { recursive: true })

const missing = []
for (const name of ["ffmpeg.exe", "ffprobe.exe"]) {
  const src = path.join(SRC, name)
  const dst = path.join(DST, name)
  if (!fs.existsSync(src)) { missing.push(name); continue }
  fs.copyFileSync(src, dst)
  log(`已嵌入: ${dst} (${(fs.statSync(dst).size / 1024).toFixed(0)} KB)`)
}

if (missing.length > 0) {
  fail(`精简版 ffmpeg/ffprobe 不完整: ${missing.join(", ")}；构建中止以避免运行时回退到系统完整版。`)
}
