#!/usr/bin/env node
// 构建 Web 前端：先 install，再 type-check，最后 vite build + 字体精简。
const fs = require("fs")
const path = require("path")
const { spawnSync } = require("child_process")

const ROOT = path.resolve(__dirname, "..")
const WEB = path.join(ROOT, "Web")

function log(m) { process.stdout.write(`[web] ${m}\n`) }
function run(cmd, args) {
  const r = spawnSync(cmd, args, { cwd: WEB, stdio: "inherit", shell: true })
  if (r.status !== 0) {
    process.stderr.write(`[web] ${cmd} ${args.join(" ")} 退出码 ${r.status}\n`)
    process.exit(r.status || 1)
  }
}

if (!fs.existsSync(path.join(WEB, "node_modules"))) {
  log("安装前端依赖 (npm install --legacy-peer-deps)...")
  run("npm", ["install", "--legacy-peer-deps", "--no-audit", "--no-fund"])
} else {
  log("node_modules 已存在，跳过 install")
}

log("类型检查 (npm run type-check)...")
const tc = spawnSync("npm", ["run", "type-check"], { cwd: WEB, stdio: "inherit", shell: true })
if (tc.status !== 0) {
  process.stderr.write(`[web] 类型检查发现问题（非致命，继续构建）\n`)
}

log("构建 (npm run build)...")
run("npm", ["run", "build"])

const dist = path.join(WEB, "dist")
if (!fs.existsSync(dist) || fs.readdirSync(dist).length === 0) {
  process.stderr.write(`[web] 产物为空: ${dist}\n`)
  process.exit(1)
}
log("前端构建完成")
