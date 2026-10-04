import { beforeEach, describe, expect, it, vi } from "vitest"
import { createPinia, setActivePinia } from "pinia"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import type { Song } from "../api/types"

const { domToPngMock } = vi.hoisted(() => ({
  domToPngMock: vi.fn(async (_node: HTMLElement) => "data:image/png;base64,test"),
}))

vi.mock("modern-screenshot", () => ({ domToPng: domToPngMock }))

vi.mock("../utils/playlistTable", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../utils/playlistTable")>()
  return { ...actual, ensureSegmenter: vi.fn(async () => undefined) }
})

import { useExportImage } from "./useExportImage"

function broadcastSong(id: number, date: string, period: "noon" | "afternoon", title: string): Song {
  return {
    id,
    date,
    weekday: "",
    title,
    artist: "",
    remark: "",
    filePath: "",
    timeSlotId: null,
    createdAt: `${date}T12:00:00+08:00`,
    period,
    order: 0,
  }
}

describe("broadcast poster export", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    domToPngMock.mockClear()
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: false })))
  })

  it("uses date rows and stable noon/afternoon columns for a seven-day playlist", async () => {
    const dates = [
      "2026-09-14",
      "2026-09-15",
      "2026-09-16",
      "2026-09-17",
      "2026-09-18",
      "2026-09-19",
      "2026-09-20",
    ]
    const songs = useSongsStore()
    songs.broadcastSongs = dates.flatMap((date, index) => [
      broadcastSong(index * 2 + 1, date, "noon", index === 0 ? "Another Love Song" : `午间歌曲${index + 1}`),
      broadcastSong(index * 2 + 2, date, "afternoon", `下午歌曲${index + 1}`),
    ])

    const settings = useSettingsStore()
    settings.broadcastColumnMap = Object.fromEntries(dates.map((_, index) => [String(index + 1), "新闻"]))

    const exportStore = useExportStore()
    exportStore.simpleMode = false

    const { generateImage } = useExportImage()
    await generateImage(dates, "broadcast")

    const container = domToPngMock.mock.calls[0][0] as HTMLElement
    const grid = container.querySelector<HTMLElement>(".ex-poster-grid.broadcast")
    expect(grid).not.toBeNull()
    expect(grid?.style.gridTemplateColumns).toBe("140px repeat(2, 1fr)")

    const cells = [...(grid?.querySelectorAll<HTMLElement>(":scope > .ex-poster-cell") ?? [])]
    expect(cells).toHaveLength(3 + dates.length * 3)
    expect(cells.slice(0, 3).map((cell) => cell.textContent?.trim())).toEqual(["", "中午", "下午"])
    expect(grid?.querySelector(".song-title")?.textContent).toBe("Another Love Song")

    const injectedCss = container.querySelector("style")?.textContent ?? ""
    expect(injectedCss).toContain("overflow-wrap: break-word")
    expect(injectedCss).not.toMatch(/\.song-title\s*\{[^}]*overflow-wrap:\s*anywhere/s)
  })
})
