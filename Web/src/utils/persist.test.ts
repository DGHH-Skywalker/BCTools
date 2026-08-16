import { describe, it, expect, beforeEach } from "vitest"
import { STORAGE_KEYS, readRaw, writeRaw, readJSON, writeJSON } from "@/utils/persist"

describe("persist storage with legacy key migration", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("reads the new prefixed key when present", () => {
    localStorage.setItem("bctools.locale", "en")
    expect(readRaw(STORAGE_KEYS.locale)).toBe("en")
  })

  it("falls back to the legacy key so existing users keep their settings", () => {
    // 老版本写的键，没有 bctools. 前缀
    localStorage.setItem("export-advanced-settings", '{"simpleMode":true}')
    const got = readJSON<{ simpleMode: boolean }>(STORAGE_KEYS.exportAdvanced)
    expect(got).toEqual({ simpleMode: true })
  })

  it("migrates the legacy value to the new key on first read", () => {
    localStorage.setItem("export-advanced-settings", '{"simpleMode":true}')
    readJSON(STORAGE_KEYS.exportAdvanced)
    expect(localStorage.getItem("bctools.export-advanced-settings")).toBe('{"simpleMode":true}')
    // 旧键保留，便于用户回退到旧版本
    expect(localStorage.getItem("export-advanced-settings")).toBe('{"simpleMode":true}')
  })

  it("prefers the new key over a stale legacy key", () => {
    localStorage.setItem("export-advanced-settings", '{"simpleMode":false}')
    localStorage.setItem("bctools.export-advanced-settings", '{"simpleMode":true}')
    expect(readJSON<any>(STORAGE_KEYS.exportAdvanced)).toEqual({ simpleMode: true })
  })

  it("returns null for missing keys instead of throwing", () => {
    expect(readRaw(STORAGE_KEYS.locale)).toBeNull()
    expect(readJSON(STORAGE_KEYS.exportAdvanced)).toBeNull()
  })

  it("returns null on corrupted JSON rather than throwing", () => {
    localStorage.setItem("bctools.export-advanced-settings", "{not json")
    expect(readJSON(STORAGE_KEYS.exportAdvanced)).toBeNull()
  })

  it("round-trips through the new key", () => {
    writeJSON(STORAGE_KEYS.exportAdvanced, { simpleMode: true, blur: 12 })
    expect(readJSON<any>(STORAGE_KEYS.exportAdvanced)).toEqual({ simpleMode: true, blur: 12 })
    expect(localStorage.getItem("bctools.export-advanced-settings")).toBeTruthy()
  })

  it("writes go to the prefixed key, never the legacy one", () => {
    writeRaw(STORAGE_KEYS.locale, "en")
    expect(localStorage.getItem("bctools.locale")).toBe("en")
    expect(localStorage.getItem("locale")).toBeNull()
  })
})
