#!/usr/bin/env node
// 清理所有构建产物：前端 dist、embed/*、Go bin、dist/、installer 产物、binembed。
const fs = require("fs")
const path = require("path")

const ROOT = path.resolve(__dirname, "..")
const targets = [
  "Web/dist",
  "GoServer/embed/dist",
  "GoServer/embed/um-react",
  "GoServer/internal/binembed/bin",
  "GoServer/resource.syso",
  "dist",
  "installer/obj",
  "installer/setup.exe",
]

function rm(p) {
  if (!fs.existsSync(p)) return
  fs.rmSync(p, { recursive: true, force: true })
  process.stdout.write(`[clean] 删除: ${p}\n`)
}

for (const t of targets) rm(path.join(ROOT, t))

// 安装程序 res/ 里复制的 exe 也清理
const installerRes = path.join(ROOT, "installer", "res")
for (const name of ["bctools.exe", "bctool_dev.exe"]) {
  const p = path.join(installerRes, name)
  if (fs.existsSync(p)) {
    try { fs.unlinkSync(p); process.stdout.write(`[clean] 删除: ${p}\n`) } catch {}
  }
}
