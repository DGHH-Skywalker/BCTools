#!/usr/bin/env node
// 构建完成后按项目实际用字精简 public/fonts 下的字体文件
const fs = require("fs")
const path = require("path")
const Fontmin = require("fontmin")

const PROJECT_ROOT = path.resolve(__dirname, "..")
const PUBLIC_FONTS_DIR = path.join(PROJECT_ROOT, "public", "fonts")
const DIST_FONTS_DIR = path.join(PROJECT_ROOT, "dist", "fonts")

// 安装程序中使用的固定文案（与 installer/src 保持一致）
const INSTALLER_TEXT = [
  "小播点歌工具",
  "预览版",
  "安装",
  "选择安装位置",
  "请选择要安装「小播点歌工具」的文件夹",
  "浏览",
  "上一步",
  "正在安装",
  "安装完成",
  "您现在可以立即运行小播点歌工具",
  "立即运行",
  "完成",
  "确认卸载",
  "此操作将删除小播点歌工具及其本地数据。\n是否继续？",
  "卸载",
  "取消",
  "正在卸载",
  "正在准备卸载…",
  "正在查找安装位置…",
  "正在停止运行中的程序…",
  "正在删除程序文件…",
  "正在清理本地数据…",
  "正在清理启动项…",
  "正在删除快捷方式…",
  "正在清理注册表…",
  "卸载完成",
  "感谢您使用小播点歌工具",
].join("")

const TEXT_FILE_EXTS = new Set([
  ".html",
  ".js",
  ".mjs",
  ".cjs",
  ".ts",
  ".tsx",
  ".vue",
  ".css",
  ".scss",
  ".json",
  ".md",
  ".ini",
  ".txt",
  ".cpp",
  ".h",
  ".rc",
])

function isTextFile(filePath) {
  return TEXT_FILE_EXTS.has(path.extname(filePath).toLowerCase())
}

function* walk(dir) {
  if (!fs.existsSync(dir)) return
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      // 跳过 node_modules 与依赖目录
      if (entry.name === "node_modules" || entry.name === ".git") continue
      yield* walk(full)
    } else if (entry.isFile()) {
      yield full
    }
  }
}

function collectChars(dirs) {
  const set = new Set()
  for (const dir of dirs) {
    for (const file of walk(dir)) {
      if (!isTextFile(file)) continue
      try {
        const text = fs.readFileSync(file, "utf-8")
        for (const ch of text) {
          // 保留基本多文种平面内有实际意义的字符
          const cp = ch.codePointAt(0)
          if (cp >= 0x20 || ch === "\n" || ch === "\r" || ch === "\t") {
            set.add(ch)
          }
        }
      } catch (err) {
        // 忽略无法读取的文件
      }
    }
  }
  return Array.from(set).join("")
}

function main() {
  if (!fs.existsSync(PUBLIC_FONTS_DIR)) {
    console.log("[subset-fonts] public/fonts 不存在，跳过")
    process.exit(0)
  }

  // 1. 从构建产物、源码、安装程序源码收集实际用字
  const scanDirs = [
    path.join(PROJECT_ROOT, "dist"),
    path.join(PROJECT_ROOT, "src"),
    path.join(PROJECT_ROOT, "..", "installer", "src"),
  ]
  const projectText = collectChars(scanDirs) + INSTALLER_TEXT

  if (!projectText) {
    console.log("[subset-fonts] 未收集到文本，跳过")
    process.exit(0)
  }

  // 2. 确保输出目录存在
  if (!fs.existsSync(DIST_FONTS_DIR)) {
    fs.mkdirSync(DIST_FONTS_DIR, { recursive: true })
  }

  // 3. 对 public/fonts 下每种字体做子集化
  const fontFiles = fs
    .readdirSync(PUBLIC_FONTS_DIR)
    .filter((f) => /\.(ttf|TTF)$/i.test(f))

  if (fontFiles.length === 0) {
    console.log("[subset-fonts] 未找到字体文件，跳过")
    process.exit(0)
  }

  let pending = fontFiles.length
  let failed = false

  for (const file of fontFiles) {
    const src = path.join(PUBLIC_FONTS_DIR, file)
    const originalSize = fs.statSync(src).size

    const fontmin = new Fontmin()
      .src(src)
      .use(Fontmin.glyph({ text: projectText, hinting: false }))
      .dest(DIST_FONTS_DIR)

    fontmin.run((err, files) => {
      if (err) {
        console.error(`[subset-fonts] ${file} 精简失败:`, err.message)
        failed = true
      } else {
        const outFile = files.find((f) => f.path.endsWith(path.extname(file)))
        const outSize = outFile ? outFile.contents.length : 0
        const ratio = originalSize ? ((1 - outSize / originalSize) * 100).toFixed(1) : 0
        console.log(
          `[subset-fonts] ${file}: ${(originalSize / 1024).toFixed(1)} KB -> ${(
            outSize / 1024
          ).toFixed(1)} KB (减少 ${ratio}%)`
        )
      }
      pending--
      if (pending === 0) {
        process.exit(failed ? 1 : 0)
      }
    })
  }
}

main()
