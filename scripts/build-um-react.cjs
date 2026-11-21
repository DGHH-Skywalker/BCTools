#!/usr/bin/env node
// 构建 um-react (Unlock Music) 子模块并复制到 GoServer/embed/um-react/。
//
// 关键点：
//   - um-react 用 pnpm（包名带 workspace）。无管理员权限时走 corepack pnpm。
//   - 复制后还要在产物的 index.html 上：
//       1) 把路由 base 改成 /um-react/
//       2) 注入桥接脚本 um-react-bridge.js
//
// 用法：node scripts/build-um-react.cjs [--skip-install] [--skip-build]
const fs = require("fs")
const path = require("path")
const { spawnSync } = require("child_process")

const ROOT = path.resolve(__dirname, "..")
const UM = path.join(ROOT, "Web", "um-react")
const DIST = path.join(UM, "dist")
const EMBED = path.join(ROOT, "GoServer", "embed", "um-react")
const BRIDGE_SRC = path.join(ROOT, "Web", "scripts", "um-react-bridge.js")

const args = new Set(process.argv.slice(2))
const skipInstall = args.has("--skip-install")
const skipBuild = args.has("--skip-build")

function log(m) { process.stdout.write(`[um-react] ${m}\n`) }
function fail(m) { process.stderr.write(`[um-react] ${m}\n`); process.exit(1) }

function run(cmd, opts = {}) {
  const r = spawnSync(cmd, opts.args || [], {
    cwd: opts.cwd,
    stdio: "inherit",
    shell: opts.shell !== false,
    env: { ...process.env, ...(opts.env || {}) },
  })
  if (r.status !== 0) {
    fail(`${cmd} ${(opts.args || []).join(" ")} 退出码 ${r.status}`)
  }
}

function findPnpm() {
  const packageManager = JSON.parse(fs.readFileSync(path.join(UM, "package.json"), "utf8")).packageManager
  if (!/^pnpm@[0-9]+\.[0-9]+\.[0-9]+$/.test(packageManager || "")) fail("um-react 未固定 pnpm 版本")
  const corepack = spawnSync("corepack", [packageManager, "--version"], { cwd: UM, shell: true, encoding: "utf8" })
  if (corepack.status === 0) return { cmd: "corepack", args: [packageManager] }
  const installed = spawnSync("pnpm", ["--version"], { cwd: UM, shell: true, encoding: "utf8" })
  if (installed.status === 0 && installed.stdout.trim() === packageManager.slice(5)) return { cmd: "pnpm", args: [] }
  fail("需要 " + packageManager + " 或 Corepack")
}

if (!fs.existsSync(path.join(UM, "package.json"))) {
  fail(`um-react 子模块不存在: ${UM}\n请先 git submodule update --init --recursive`)
}

const pnpm = findPnpm()
const pnpmArgs = (...rest) => [...pnpm.args, ...rest]

if (!skipInstall && !fs.existsSync(path.join(UM, "node_modules"))) {
  log("安装 um-react 依赖 (pnpm install)...")
  run(pnpm.cmd, { args: pnpmArgs("install", "--frozen-lockfile"), cwd: UM })
} else if (fs.existsSync(path.join(UM, "node_modules"))) {
  log("node_modules 已存在，跳过 install")
}

if (!skipBuild) {
  log("构建 um-react (pnpm build)...")
  run(pnpm.cmd, { args: pnpmArgs("exec", "tsc", "-p", "tsconfig.prod.json"), cwd: UM })
  run(pnpm.cmd, { args: pnpmArgs("exec", "vite", "build"), cwd: UM })
  run(process.execPath, { args: ["scripts/write-version.mjs"], cwd: UM, shell: false })
}

if (!fs.existsSync(path.join(DIST, "index.html"))) {
  fail(`um-react 产物为空: ${DIST}`)
}

// 改路由 base
const idxPath = path.join(DIST, "index.html")
let html = fs.readFileSync(idxPath, "utf8")
const patched = html.replace("window.BASE_URL = '/'", "window.BASE_URL = '/um-react/'")
if (patched === html) log("  警告：未在 index.html 找到 window.BASE_URL，路由 base 未修改")
else { fs.writeFileSync(idxPath, patched, "utf8"); log("  已设置路由 base 为 /um-react/") }

// 注入桥接脚本
if (fs.existsSync(BRIDGE_SRC)) {
  fs.copyFileSync(BRIDGE_SRC, path.join(DIST, "um-react-bridge.js"))
  html = fs.readFileSync(idxPath, "utf8")
  const tag = '<script src="./um-react-bridge.js"></script>'
  if (!html.includes(tag)) {
    const newHtml = html.replace("</head>", tag + "</head>")
    fs.writeFileSync(idxPath, newHtml, "utf8")
  }
  log("  已注入 um-react 桥接脚本")
}

// 复制到 embed
if (fs.existsSync(EMBED)) fs.rmSync(EMBED, { recursive: true, force: true })
fs.mkdirSync(EMBED, { recursive: true })
fs.cpSync(DIST, EMBED, { recursive: true })
log(`已复制到: ${EMBED}`)
