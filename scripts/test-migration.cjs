#!/usr/bin/env node
// 端到端测试：导出 -> 导入（merge / replace / 错误处理）
const fs = require("fs")
const path = require("path")
const { execSync, spawn } = require("child_process")

const PORT = 1799
const BASE = "C:\\Users\\pc\\Desktop"

async function call(method, path, body) {
  const res = await fetch(`http://localhost:${PORT}${path}`, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let json
  try { json = JSON.parse(text) } catch { json = { raw: text } }
  return { status: res.status, body: json }
}

async function sleep(ms) { return new Promise(r => setTimeout(r, ms)) }

async function main() {
  console.log("=== 1. export files ===")
  const exp = await call("POST", "/api/migration/export", {
    scope: "files",
    targetDir: BASE + "\\mig_full",
  })
  console.log("HTTP", exp.status, JSON.stringify(exp.body))

  await sleep(500)

  // 找子目录
  const fullDir = BASE + "\\mig_full"
  if (!fs.existsSync(fullDir)) { console.error("no export dir"); process.exit(1) }
  const entries = fs.readdirSync(fullDir).filter(e => e.includes("广播站点歌工具数据备份"))
  if (entries.length === 0) { console.error("no backup subdir"); process.exit(1) }
  const backupName = entries.sort().pop()
  const backupDir = `${fullDir}\\${backupName}`
  console.log("backup:", backupDir)

  // 列目录
  const files = fs.readdirSync(backupDir)
  console.log("contents:", files.slice(0, 5).join(", "), files.length > 5 ? `...(${files.length} total)` : "")

  console.log("=== 2. import (merge) ===")
  let r = await call("POST", "/api/migration/import", { backupDir, mode: "merge" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 3. import (merge again) ===")
  r = await call("POST", "/api/migration/import", { backupDir, mode: "merge" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 4. import (replace) ===")
  r = await call("POST", "/api/migration/import", { backupDir, mode: "replace" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 5. import (invalid dir) ===")
  r = await call("POST", "/api/migration/import", { backupDir: BASE + "\\nonexistent_xyz", mode: "merge" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 6. import (bad mode) ===")
  r = await call("POST", "/api/migration/import", { backupDir, mode: "invalid" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 7. import (missing data.json) ===")
  fs.mkdirSync(BASE + "\\empty_dir_xyz", { recursive: true })
  r = await call("POST", "/api/migration/import", { backupDir: BASE + "\\empty_dir_xyz", mode: "merge" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== 8. JSON export ===")
  const jsonRes = await fetch(`http://localhost:${PORT}/api/migration/export`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scope: "json" }),
  })
  console.log("HTTP", jsonRes.status, "size:", (await jsonRes.text()).length)

  console.log("=== 9. schema mismatch ===")
  const wrongDir = BASE + "\\wrong_schema"
  fs.mkdirSync(wrongDir, { recursive: true })
  fs.writeFileSync(path.join(wrongDir, "data.json"), JSON.stringify({
    schemaVersion: 999,
    appVersion: "0.0.0",
    exportedAt: new Date().toISOString(),
    dormSongs: [], broadcastSongs: [], settings: {},
  }))
  r = await call("POST", "/api/migration/import", { backupDir: wrongDir, mode: "merge" })
  console.log("HTTP", r.status, JSON.stringify(r.body))

  console.log("=== ALL DONE ===")
}

main().catch(e => { console.error("ERR", e); process.exit(1) })
