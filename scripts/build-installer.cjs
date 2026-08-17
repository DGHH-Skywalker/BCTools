#!/usr/bin/env node
// 构建 C++ 安装程序（installer/）。需要 MinGW-w64 提供 g++ 和 make。
//
// 流程：复制资源 → 生成版本头 → 调用 make → 把 setup.exe 复制到 dist/。
// 没有 MinGW 时直接报错退出，不影响 npm run build 主链路。
const fs = require("fs")
const path = require("path")
const { spawnSync } = require("child_process")

const ROOT = path.resolve(__dirname, "..")
const INSTALLER = path.join(ROOT, "installer")
const INSTALLER_RES = path.join(INSTALLER, "res")
const BE = path.join(ROOT, "GoServer")
const FE = path.join(ROOT, "Web")
const DIST = path.join(ROOT, "dist")
const VERSION_FILE = path.join(BE, "internal", "version", "version.go")
const VERSION_HEADER = path.join(INSTALLER, "src", "version.h")

function log(m) { process.stdout.write(`[installer] ${m}\n`) }
function fail(m) { process.stderr.write(`[installer] ${m}\n`); process.exit(1) }

function findBinary(name) {
  // 优先 PATH
  const r = spawnSync(name, ["--version"], { stdio: "pipe", encoding: "utf8" })
  if (r.status === 0) return name
  // 常见 MinGW 路径
  const candidates = [
    "C:/Tools/mingw64/mingw64/bin",
    "C:/mingw64/bin",
    "C:/msys64/mingw64/bin",
    "C:/Program Files/mingw-w64/x86_64-8.1.0-posix-seh-rt_v6-rev0/mingw64/bin",
  ]
  const ext = process.platform === "win32" ? ".exe" : ""
  for (const c of candidates) {
    if (fs.existsSync(path.join(c, name + ext))) {
      process.env.PATH = c + path.delimiter + process.env.PATH
      return name
    }
  }
  return null
}

if (!findBinary("g++")) {
  log("未找到 g++：跳过安装程序构建（不影响 bctools.exe 主产物）")
  log("  → 想编译 BCTools-Setup.exe，请安装 MinGW-w64 并把 bin 加到 PATH")
  process.exit(0)
}

if (!findBinary("make") && !findBinary("mingw32-make")) {
  log("未找到 make / mingw32-make：跳过安装程序构建")
  log("  → 想编译 BCTools-Setup.exe，请安装 MinGW-w64")
  process.exit(0)
}
const make = findBinary("make") || "mingw32-make"

// 准备资源
fs.mkdirSync(INSTALLER_RES, { recursive: true })
const resources = [
  [path.join(BE, "assets/icon-bc.ico"), path.join(INSTALLER_RES, "icon-bc.ico")],
  [path.join(FE, "public/logo.png"), path.join(INSTALLER_RES, "logo.png")],
  [path.join(FE, "dist/fonts/江西拙楷3.0.ttf"), path.join(INSTALLER_RES, "font_jiangxi_zhuokai.ttf")],
  [path.join(FE, "dist/fonts/方正颜宋简体.ttf"), path.join(INSTALLER_RES, "font_fangzheng_yansong.ttf")],
  [path.join(DIST, "bctools.exe"), path.join(INSTALLER_RES, "bctools.exe")],
]
for (const [src, dst] of resources) {
  if (!fs.existsSync(src)) { log(`  警告：资源缺失 ${src}，跳过`); continue }
  fs.copyFileSync(src, dst)
  log(`  资源: ${path.basename(dst)}`)
}

// 生成版本头
if (fs.existsSync(VERSION_FILE)) {
  const content = fs.readFileSync(VERSION_FILE, "utf8")
  const m = content.match(/const\s+Version\s+=\s+"([^"]+)"/)
  if (m) {
    fs.writeFileSync(VERSION_HEADER, `#pragma once\n\n#define APP_VERSION L"${m[1]}"\n`, "utf8")
    log(`  版本头: ${m[1]}`)
  }
}

log(`构建 (${make})...`)
const build = spawnSync(make, [], { cwd: INSTALLER, stdio: "inherit", shell: true })
if (build.status !== 0) fail(`make 退出码 ${build.status}`)

const setupSrc = path.join(INSTALLER, "setup.exe")
if (!fs.existsSync(setupSrc)) fail(`安装程序产物不存在: ${setupSrc}`)

fs.mkdirSync(DIST, { recursive: true })
const setupDst = path.join(DIST, "BCTools-Setup.exe")
fs.copyFileSync(setupSrc, setupDst)
log(`已输出: ${setupDst} (${(fs.statSync(setupDst).size / 1024).toFixed(0)} KB)`)
