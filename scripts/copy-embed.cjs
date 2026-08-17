#!/usr/bin/env node
// 把 Web 前端构建产物复制到 GoServer/embed/dist，供 //go:embed 打进 Go 二进制。
//
// 依赖约定：
//   - 源：<repo>/Web/dist  （vite build 产出）
//   - 目标：<repo>/GoServer/embed/dist  （go:embed embed/dist/* 读取的位置）
//
// 行为：
//   - 目标目录存在时先清空再复制（避免上次构建残留）
//   - 源目录不存在则报错退出（多半是 build:web 没跑）
//   - 复制过程保留相对路径，过滤掉源目录里的空目录

const fs = require("fs")
const path = require("path")

const REPO_ROOT = path.resolve(__dirname, "..")
const SRC = path.join(REPO_ROOT, "Web", "dist")
const DST = path.join(REPO_ROOT, "GoServer", "embed", "dist")

function log(msg) {
  process.stdout.write(`[copy-embed] ${msg}\n`)
}

function fail(msg) {
  process.stderr.write(`[copy-embed] ${msg}\n`)
  process.exit(1)
}

function rimraf(dir) {
  if (!fs.existsSync(dir)) return
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name)
    if (entry.isDirectory()) rimraf(p)
    else fs.unlinkSync(p)
  }
  fs.rmdirSync(dir)
}

function copyDir(src, dst) {
  fs.mkdirSync(dst, { recursive: true })
  for (const entry of fs.readdirSync(src, { withFileTypes: true })) {
    const s = path.join(src, entry.name)
    const d = path.join(dst, entry.name)
    if (entry.isDirectory()) copyDir(s, d)
    else fs.copyFileSync(s, d)
  }
}

if (!fs.existsSync(SRC)) {
  fail(`源目录不存在: ${SRC}\n请先运行 npm run build:web`)
}

log(`清空目标: ${DST}`)
rimraf(DST)

log(`复制 ${SRC} -> ${DST}`)
copyDir(SRC, DST)

let totalBytes = 0
let fileCount = 0
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(p)
    else {
      totalBytes += fs.statSync(p).size
      fileCount++
    }
  }
}
walk(DST)
log(`完成：${fileCount} 个文件，${(totalBytes / 1024).toFixed(0)} KB`)
