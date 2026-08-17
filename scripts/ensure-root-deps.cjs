#!/usr/bin/env node
// 首次跑 npm run dev / build 时确保根 node_modules 装好。
//
// 设计：把"npm install"作为 dev / build 入口的前置钩子（predev / prebuild），
// 用户 clone 后只需要 node + go，不需要先 cd 到根手动 npm install。
//
// 已存在 node_modules + 所有 devDependencies 时直接退出（幂等）。
const fs = require("fs")
const path = require("path")
const { spawnSync } = require("child_process")

const ROOT = path.resolve(__dirname, "..")
const NODE_MODULES = path.join(ROOT, "node_modules")
const PKG = JSON.parse(fs.readFileSync(path.join(ROOT, "package.json"), "utf8"))

const devDeps = Object.keys(PKG.devDependencies || {})
const hasAll = (pkgs) => pkgs.every(p => fs.existsSync(path.join(NODE_MODULES, p)))

if (fs.existsSync(NODE_MODULES) && hasAll(devDeps)) {
  process.exit(0)
}

process.stdout.write("[deps] 首次运行：安装根 devDependencies...\n")
const r = spawnSync("npm", ["install", "--no-audit", "--no-fund"], {
  cwd: ROOT,
  stdio: "inherit",
  shell: true,
})
if (r.status !== 0) {
  process.stderr.write(`[deps] npm install 失败，退出码 ${r.status}\n`)
  process.exit(r.status || 1)
}
