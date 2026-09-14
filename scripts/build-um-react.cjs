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
  // 1) 直接 pnpm（全局安装）
  const r = spawnSync("pnpm", ["--version"], { stdio: "pipe", encoding: "utf8", shell: true })
  if (r.status === 0) return { cmd: "pnpm", args: [] }
  // 2) corepack pnpm（不需要管理员写到 Program Files）。
  const r2 = spawnSync("corepack", ["pnpm", "--version"], {
    stdio: "pipe",
    encoding: "utf8",
    shell: true,
  })
  if (r2.status === 0) return { cmd: "corepack", args: ["pnpm"] }
  fail("未找到 pnpm。请先 npm i -g pnpm，或保证 corepack 可用。")
}

if (!fs.existsSync(path.join(UM, "package.json"))) {
  fail(`um-react 子模块不存在: ${UM}\n请先 git submodule update --init --recursive`)
}

const pnpm = findPnpm()
const pnpmArgs = (...rest) => [...pnpm.args, ...rest]

// um-react 自己的 package.json 里 build 末尾会再调一次 `pnpm build:finalize`。
// 我们这里只暴露 corepack pnpm 给 PATH，um-react 的脚本里 `pnpm ...` 就解析
// 不到。解决办法：把 corepack 自身的路径加到 PATH，并塞一个 pnpm.cmd shim 到
// 当前 PATH 里的可写目录。
function ensurePnpmShim() {
  if (pnpm.cmd === "pnpm") return {} // 已有真 pnpm，无需 shim
  // 找 corepack 所在目录（C:\Program Files\nodejs）
  const which = spawnSync("corepack", [], { stdio: "pipe", encoding: "utf8", shell: true })
  // 用 cmd where 找
  const where = spawnSync("cmd", ["/c", "where", "corepack"], {
    stdio: "pipe",
    encoding: "utf8",
    shell: true,
  })
  const corepackPath = (where.stdout || "").split(/\r?\n/)[0]?.trim()
  if (!corepackPath) return {}
  const corepackDir = path.dirname(corepackPath)
  // 写一个 pnpm.cmd 到 .corepack-shims/
  const shimDir = path.join(ROOT, ".corepack-shims")
  fs.mkdirSync(shimDir, { recursive: true })
  const shim = path.join(shimDir, "pnpm.cmd")
  fs.writeFileSync(
    shim,
    `@echo off\r\n"${corepackPath}" pnpm %*\r\n`,
    "utf8",
  )
  // 同时也写个 pnpm（无扩展名）给 bash 找
  fs.writeFileSync(
    path.join(shimDir, "pnpm"),
    `#!/bin/sh\nexec "${corepackPath}" pnpm "$@"\n`,
    "utf8",
  )
  log(`  pnpm shim: ${shimDir}`)
  return { PATH: shimDir + path.delimiter + (process.env.PATH || "") }
}
const extraEnv = ensurePnpmShim()

if (!skipInstall && !fs.existsSync(path.join(UM, "node_modules"))) {
  log("安装 um-react 依赖 (pnpm install)...")
  run(pnpm.cmd, { args: pnpmArgs("install", "--no-frozen-lockfile"), cwd: UM, env: extraEnv })
} else if (fs.existsSync(path.join(UM, "node_modules"))) {
  log("node_modules 已存在，跳过 install")
}

if (!skipBuild) {
  log("构建 um-react (pnpm build)...")
  run(pnpm.cmd, { args: pnpmArgs("build"), cwd: UM, env: extraEnv })
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
