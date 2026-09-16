#!/usr/bin/env node
// Compile the Inno Setup package from the already-built Go executable.
const fs = require("fs")
const path = require("path")
const { spawnSync } = require("child_process")

const ROOT = path.resolve(__dirname, "..")
const DIST = path.join(ROOT, "dist")
const SCRIPT = path.join(ROOT, "installer", "BCTools.iss")
const VERSION_FILE = path.join(ROOT, "GoServer", "internal", "version", "version.go")
const OUTPUT = path.join(DIST, "BCTools-Setup.exe")

function fail(message) {
  process.stderr.write(`[installer] ${message}\n`)
  process.exit(1)
}

function findISCC() {
  const candidates = [
    process.env.ISCC_PATH,
    "C:/Program Files (x86)/Inno Setup 6/ISCC.exe",
    "C:/Program Files/Inno Setup 6/ISCC.exe",
    process.env.LOCALAPPDATA && path.join(process.env.LOCALAPPDATA, "Programs", "Inno Setup 6", "ISCC.exe"),
  ].filter(Boolean)
  return candidates.find((candidate) => fs.existsSync(candidate))
}

if (!fs.existsSync(path.join(DIST, "bctools.exe"))) {
  fail("dist/bctools.exe 不存在，请先运行 npm run build:server")
}
if (!fs.existsSync(SCRIPT)) fail(`Inno Setup 脚本不存在: ${SCRIPT}`)

const versionSource = fs.readFileSync(VERSION_FILE, "utf8")
const version = versionSource.match(/const\s+Version\s*=\s*"([^"]+)"/)?.[1]
if (!version) fail("无法从 GoServer/internal/version/version.go 读取版本号")

const iscc = findISCC()
if (!iscc) {
  fail("未找到 Inno Setup 6 编译器 ISCC.exe。请安装 Inno Setup 6，或设置 ISCC_PATH。")
}

fs.mkdirSync(DIST, { recursive: true })
process.stdout.write(`[installer] 使用 ${iscc}\n`)
process.stdout.write(`[installer] 构建 BCTools ${version}\n`)
const result = spawnSync(iscc, [`/DMyAppVersion=${version}`, SCRIPT], {
  cwd: ROOT,
  stdio: "inherit",
  windowsHide: true,
})
if (result.error) fail(result.error.message)
if (result.status !== 0) fail(`ISCC 退出码 ${result.status}`)
if (!fs.existsSync(OUTPUT)) fail(`安装程序产物不存在: ${OUTPUT}`)

process.stdout.write(`[installer] 已输出: ${OUTPUT} (${Math.round(fs.statSync(OUTPUT).size / 1024)} KB)\n`)
