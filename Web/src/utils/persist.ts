// localStorage 单一入口。
//
// 此前读写散落在 stores/export.ts 与 i18n/index.ts，key 命名也不统一
// （"export-advanced-settings"、"locale"）。统一加 "bctools." 前缀，
// 避免与同源下的其他页面（如 /um-react/）互相踩键。
//
// 重要：换前缀会让老用户已保存的设置读不到。因此读取时若新 key 不存在，
// 回落到旧 key 并顺带迁移过去，绝不能让用户的导出配置凭空消失。

const PREFIX = "bctools."

/** 各持久化项的 key：新 key 与需要兼容读取的历史 key。 */
export const STORAGE_KEYS = {
  exportAdvanced: { key: PREFIX + "export-advanced-settings", legacy: ["export-advanced-settings"] },
  locale: { key: PREFIX + "locale", legacy: ["locale"] },
} as const

type StorageSpec = { key: string; legacy: readonly string[] }

/** 读原始字符串。新 key 缺失时回落到历史 key，并迁移到新 key。 */
export function readRaw(spec: StorageSpec): string | null {
  try {
    const hit = localStorage.getItem(spec.key)
    if (hit !== null) return hit
    for (const old of spec.legacy) {
      const legacyHit = localStorage.getItem(old)
      if (legacyHit !== null) {
        // 迁移：写入新 key，保留旧 key 不删，便于回退到旧版本。
        try {
          localStorage.setItem(spec.key, legacyHit)
        } catch {
          // 配额或隐私模式，忽略——本次仍按旧值返回。
        }
        return legacyHit
      }
    }
  } catch {
    // localStorage 不可用（隐私模式 / 被策略禁用），当作没有存储。
  }
  return null
}

/** 写原始字符串。存储不可用时静默忽略——持久化是增强，不是必需。 */
export function writeRaw(spec: StorageSpec, value: string): void {
  try {
    localStorage.setItem(spec.key, value)
  } catch {
    // ignore
  }
}

/** 读 JSON。解析失败或不存在时返回 null，由调用方决定默认值。 */
export function readJSON<T>(spec: StorageSpec): T | null {
  const raw = readRaw(spec)
  if (raw === null) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

/** 写 JSON。 */
export function writeJSON(spec: StorageSpec, value: unknown): void {
  try {
    writeRaw(spec, JSON.stringify(value))
  } catch {
    // 循环引用等序列化失败，忽略。
  }
}
