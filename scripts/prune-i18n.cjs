#!/usr/bin/env node
// 删除未引用的 i18n 词条；保留通过 index 访问的数组类型（weekdays[] /
// weekdaysShort[] / broadcast.columns[]）。
//
// 用法：node scripts/prune-i18n.cjs
//  实际写入；运行前请备份或 git diff 自查。

const fs = require("fs")
const path = require("path")

const ROOT = path.join(__dirname, "..", "Web", "src")
const LOCALES = ["zh-CN.json", "en.json"]

function walk(dir, files = []) {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) walk(p, files)
    else if (/\.(ts|vue)$/.test(e.name)) files.push(p)
  }
  return files
}

function flatten(obj, prefix = "") {
  const out = []
  for (const k of Object.keys(obj || {})) {
    const v = obj[k]
    const key = prefix ? prefix + "." + k : k
    if (Array.isArray(v)) {
      // 数组类型：标记为「保留」——它们按 index 取，不是按字符串 key 查
      out.push({ key, kind: "array" })
    } else if (typeof v === "object" && v !== null) {
      out.push(...flatten(v, key))
    } else if (typeof v === "string") {
      out.push({ key, kind: "string" })
    }
  }
  return out
}

const files = walk(ROOT).filter(
  f => !f.includes("locales/") && !f.endsWith(".test.ts"),
)
// 全部内容拼成一份大字符串，加速正则匹配
const haystack = files.map(f => fs.readFileSync(f, "utf8")).join("\n")

const isUsed = key => {
  const re = new RegExp(`t\\(\\s*["']${key.replace(/\./g, "\\.")}["']`)
  return re.test(haystack)
}

for (const file of LOCALES) {
  const fp = path.join(ROOT, "locales", file)
  const data = JSON.parse(fs.readFileSync(fp, "utf8"))
  const before = JSON.stringify(data).length
  const flat = flatten(data)
  const toRemove = []
  for (const { key, kind } of flat) {
    if (kind === "array") continue // 数组保留
    if (!isUsed(key)) toRemove.push(key)
  }
  console.log(`[${file}] 待删除 ${toRemove.length} 个 key`)
  for (const key of toRemove) {
    const parts = key.split(".")
    let cur = data
    for (let i = 0; i < parts.length - 1; i++) {
      cur = cur?.[parts[i]]
      if (!cur) break
    }
    if (cur && parts[parts.length - 1] in cur) {
      delete cur[parts[parts.length - 1]]
    }
  }
  // 删完空对象（仅当父对象的所有叶子都空了）
  function pruneEmpty(obj) {
    for (const k of Object.keys(obj)) {
      if (obj[k] && typeof obj[k] === "object" && !Array.isArray(obj[k])) {
        pruneEmpty(obj[k])
        if (Object.keys(obj[k]).length === 0) delete obj[k]
      }
    }
  }
  pruneEmpty(data)
  const after = JSON.stringify(data).length
  fs.writeFileSync(fp, JSON.stringify(data, null, 2) + "\n", "utf8")
  console.log(`  ${before} → ${after} 字节（-${before - after}）`)
}
