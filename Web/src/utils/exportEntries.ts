import type { Song, TimeSlot } from "../api/types"
import { dayIndexFromDate } from "./datetime"

export interface ExportEntry {
  /** 源文件路径；空字符串表示该位置需要生成静音占位文件。 */
  source: string
  /** SD 卡上的目标文件名，形如 "01.mp3"。 */
  targetName: string
}

/**
 * 按「时段位置」而非「实际歌曲」编号，供换卡工具复制到 SD 卡。
 *
 * 每个时段恒定占用一个序号：有歌就复制歌，没歌就留空（由调用方生成静音占位）。
 * 这一点很关键——否则整天没点歌时（例如周六全天空着），后面的歌会整体前移，
 * 周日的歌会从 21 开始而不是 25，SD 卡上的曲序就和贴出来的歌单对不上了。
 *
 * 同一时段允许多首歌（拖拽排序决定先后），此时它们连续占用多个序号。
 * 指向已删除时段的孤儿歌曲会被跳过——给它们编号会把后面的全推错。
 */
export function buildExportEntries(
  dates: string[],
  songs: Song[],
  timeSlots: TimeSlot[],
): ExportEntry[] {
  const entries: ExportEntry[] = []
  let seq = 1
  const push = (source: string) => {
    entries.push({ source, targetName: `${String(seq).padStart(2, "0")}.mp3` })
    seq++
  }

  for (const date of dates) {
    const dayIdx = dayIndexFromDate(date)
    const daySlots = timeSlots
      .filter((s) => s.dayIndex === dayIdx)
      .sort((a, b) => a.order - b.order)

    for (const slot of daySlots) {
      const slotSongs = songs
        .filter((s) => s.date === date && s.timeSlotId === slot.id)
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)

      if (slotSongs.length === 0) {
        push("") // 空时段：占一个序号，交给调用方生成静音占位
        continue
      }
      for (const song of slotSongs) {
        push(song.filePath || "")
      }
    }
  }
  return entries
}

/**
 * 判断这一批条目里是否至少有一首真实歌曲。
 *
 * buildExportEntries 会为空时段生成占位，所以 entries.length 不再代表「有没有歌」：
 * 一周完全没点歌时它会返回一整周的占位。那种情况下不该往 SD 卡写满静音文件。
 */
export function hasAnySong(entries: ExportEntry[]): boolean {
  return entries.some((e) => e.source !== "")
}
