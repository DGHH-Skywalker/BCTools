import { ref, readonly } from "vue"

export interface AppConfig {
  languageSwitchEnabled: boolean
  themeColor: string
  importXlsxEnabled: boolean
  exportXlsxEnabled: boolean
}

const config = ref<AppConfig>({
  languageSwitchEnabled: false,
  themeColor: "#0086C3",
  importXlsxEnabled: false,
  exportXlsxEnabled: false,
})

let loaded = false

function parseIni(text: string): Record<string, string> {
  const result: Record<string, string> = {}
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith(";") || line.startsWith("#") || line.startsWith("[")) continue
    const idx = line.indexOf("=")
    if (idx < 0) continue
    const key = line.slice(0, idx).trim().toLowerCase()
    const value = line.slice(idx + 1).trim()
    result[key] = value.replace(/^["']|["']$/g, "")
  }
  return result
}

function isHexColor(v: string): boolean {
  return /^#([0-9a-fA-F]{6}|[0-9a-fA-F]{3})$/.test(v)
}

export async function loadAppConfig(): Promise<AppConfig> {
  if (loaded) return config.value
  try {
    const res = await fetch("/config.ini", { cache: "no-store" })
    if (res.ok) {
      const parsed = parseIni(await res.text())
      // 当前版本仅支持中文，强制关闭语言切换
      config.value.languageSwitchEnabled = false
      config.value.importXlsxEnabled = parsed.importxlsxenabled === "true"
      config.value.exportXlsxEnabled = parsed.exportxlsxenabled === "true"
      if (parsed.themecolor && isHexColor(parsed.themecolor)) {
        config.value.themeColor = parsed.themecolor.toLowerCase()
      }
    }
  } catch (err) {
    console.warn("Failed to load config.ini:", err)
  }
  loaded = true
  return config.value
}

export function useAppConfig() {
  return { config: readonly(config), loadAppConfig }
}
