import type { Song, TimeSlot } from "../api/types"
import { dayIndexFromDate } from "./datetime"

export interface ExportEntry {
  /** 单个源文件路径；空字符串表示该位置需要生成静音占位文件。 */
  source: string
  /**
   * 同一时段的多个源文件，按播放顺序排列。
   * 长度 > 1 时后端会用它们合并成一个 MP3（歌库里仍分开存放）。
   */
  sources?: string[]
  /** SD 卡上的目标文件名，形如 "01.mp3"。 */
  targetName: string
}

/**
 * 按「时段位置」而非「实际歌曲」编号，供换卡工具复制到 SD 卡。
 *
 * 一个时段 = 一个序号 = SD 卡上一个文件：
 *   - 有 1 首歌  -> 直接复制
 *   - 有多首歌  -> 按拖拽顺序合并成一个 MP3（播放器只会顺序播下一个文件，
 *                  合并后同一时段的几首歌才会连着放完）
 *   - 没有歌    -> 生成静音占位
 *
 * 编号必须锚定时段位置：否则整天没点歌时（例如周六全天空着），后面的歌会整体
 * 前移，周日的歌会从 21 开始而不是 25，SD 卡曲序就和贴出来的歌单对不上了。
 *
 * 指向已删除时段的孤儿歌曲会被跳过——给它们编号会把后面的全推错。
 */
export function buildExportEntries(
  dates: string[],
  songs: Song[],
  timeSlots: TimeSlot[],
): ExportEntry[] {
  const entries: ExportEntry[] = []
  let seq = 1
  const nextName = () => `${String(seq++).padStart(2, "0")}.mp3`

  for (const date of dates) {
    const dayIdx = dayIndexFromDate(date)
    const daySlots = timeSlots
      .filter((s) => s.dayIndex === dayIdx)
      .sort((a, b) => a.order - b.order)

    for (const slot of daySlots) {
      const slotSongs = songs
        .filter((s) => s.date === date && s.timeSlotId === slot.id)
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)

      const paths = slotSongs.map((s) => s.filePath || "").filter((p) => p !== "")

      if (paths.length === 0) {
        // 空时段（或该时段的歌都没有文件）：占一个序号，生成静音占位
        entries.push({ source: "", targetName: nextName() })
        continue
      }
      if (paths.length === 1) {
        entries.push({ source: paths[0], targetName: nextName() })
        continue
      }
      // 多首歌合并成一个文件，仍只占一个序号
      entries.push({ source: paths[0], sources: paths, targetName: nextName() })
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
