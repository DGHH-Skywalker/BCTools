import { describe, it, expect } from "vitest"
import { buildExportEntries } from "@/utils/exportEntries"
import type { Song, TimeSlot } from "@/api/types"

// 一周 7 天、每天 4 个时段（与当前实际配置一致：06:35 / 06:50 / 13:55 / 18:30）
const WEEK = [
  "2026-08-17", // 周一
  "2026-08-18",
  "2026-08-19",
  "2026-08-20",
  "2026-08-21",
  "2026-08-22", // 周六
  "2026-08-23", // 周日
]

function slots(): TimeSlot[] {
  const times = ["06:35", "06:50", "13:55", "18:30"]
  const out: TimeSlot[] = []
  for (let day = 1; day <= 7; day++) {
    times.forEach((time, i) => {
      out.push({ id: `d${day}-s${i + 1}`, dayIndex: day, order: i + 1, time })
    })
  }
  return out
}

function song(id: number, date: string, timeSlotId: string, order = 0): Song {
  return {
    id, date, weekday: "", title: `歌${id}`, artist: "", remark: "",
    filePath: `/songs/${id}.mp3`, timeSlotId, createdAt: "", order,
  }
}

describe("buildExportEntries slot-aligned numbering", () => {
  it("keeps Sunday starting at 25 when Saturday is entirely empty", () => {
    // 周一~周五 + 周日 都点满，周六一整天空着
    const songs: Song[] = []
    let id = 1
    for (const [dayIdx, date] of WEEK.entries()) {
      if (date === "2026-08-22") continue // 周六不点歌
      for (let s = 1; s <= 4; s++) {
        songs.push(song(id++, date, `d${dayIdx + 1}-s${s}`))
      }
    }

    const entries = buildExportEntries(WEEK, songs, slots())

    // 7 天 × 4 时段 = 28 个序号，一个不少
    expect(entries).toHaveLength(28)
    expect(entries[0].targetName).toBe("01.mp3")
    expect(entries[27].targetName).toBe("28.mp3")

    // 周六（第 6 天）占 21-24，且必须是静音占位（source 为空）
    const sat = entries.slice(20, 24)
    expect(sat.map((e) => e.targetName)).toEqual(["21.mp3", "22.mp3", "23.mp3", "24.mp3"])
    expect(sat.every((e) => e.source === "")).toBe(true)

    // 关键回归点：周日必须从 25 开始，而不是被前移到 21
    const sun = entries.slice(24, 28)
    expect(sun.map((e) => e.targetName)).toEqual(["25.mp3", "26.mp3", "27.mp3", "28.mp3"])
    expect(sun.every((e) => e.source !== "")).toBe(true)
  })

  it("emits a placeholder for a single missing slot without shifting the rest", () => {
    const songs: Song[] = []
    let id = 1
    for (const [dayIdx, date] of WEEK.entries()) {
      for (let s = 1; s <= 4; s++) {
        // 周一第 2 个时段空着
        if (dayIdx === 0 && s === 2) continue
        songs.push(song(id++, date, `d${dayIdx + 1}-s${s}`))
      }
    }

    const entries = buildExportEntries(WEEK, songs, slots())
    expect(entries).toHaveLength(28)
    // 02 是占位，03 仍然是周一第 3 个时段的歌（没被提到 02）
    expect(entries[1].source).toBe("")
    expect(entries[1].targetName).toBe("02.mp3")
    expect(entries[2].source).not.toBe("")
  })

  it("merges multiple songs in one slot into a single numbered file", () => {
    const songs = [
      song(1, WEEK[0], "d1-s1", 0),
      song(2, WEEK[0], "d1-s1", 1),
    ]
    const entries = buildExportEntries(WEEK, songs, slots())
    // 同一时段的两首歌合并成一个文件，仍只占 01 这一个序号；
    // 一周 28 个时段 -> 恒定 28 个文件。
    expect(entries).toHaveLength(28)
    expect(entries[0].targetName).toBe("01.mp3")
    expect(entries[0].sources).toEqual(["/songs/1.mp3", "/songs/2.mp3"])
    // 下一个时段仍是 02，没有被挤走
    expect(entries[1].targetName).toBe("02.mp3")
    expect(entries[1].sources).toBeUndefined()
  })

  it("orders merged songs within a slot by drag order, not by id", () => {
    const songs = [
      song(9, WEEK[0], "d1-s1", 1), // 拖到第二位
      song(3, WEEK[0], "d1-s1", 0), // 第一位
    ]
    const entries = buildExportEntries(WEEK, songs, slots())
    // 合并顺序必须跟拖拽顺序，否则 SD 卡上播放次序是错的
    expect(entries[0].sources).toEqual(["/songs/3.mp3", "/songs/9.mp3"])
  })

  it("does not set sources for a single-song slot", () => {
    const entries = buildExportEntries(WEEK, [song(1, WEEK[0], "d1-s1")], slots())
    expect(entries[0].source).toBe("/songs/1.mp3")
    expect(entries[0].sources).toBeUndefined()
  })

  it("ignores songs assigned to a deleted slot rather than misnumbering", () => {
    const songs = [
      song(1, WEEK[0], "d1-s1"),
      song(2, WEEK[0], "does-not-exist"), // 指向已删除的时段
    ]
    const entries = buildExportEntries(WEEK, songs, slots())
    expect(entries).toHaveLength(28)
    expect(entries[0].source).toBe("/songs/1.mp3")
    // 孤儿歌曲不占序号，否则会把后面全推错
    expect(entries.filter((e) => e.source === "/songs/2.mp3")).toHaveLength(0)
  })

  it("returns all placeholders when nothing is scheduled", () => {
    const entries = buildExportEntries(WEEK, [], slots())
    expect(entries).toHaveLength(28)
    expect(entries.every((e) => e.source === "")).toBe(true)
  })

  it("returns nothing when no time slots are configured", () => {
    expect(buildExportEntries(WEEK, [], [])).toHaveLength(0)
  })
})
