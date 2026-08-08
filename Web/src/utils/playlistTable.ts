import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import { Segment, useDefault } from "segmentit"
import type { Song, TimeSlot } from "../api/types"

dayjs.extend(isoWeek)

const segment = useDefault(new Segment())

export interface BuildDormTableOptions {
  title?: string | null
  songs: Song[]
  timeSlots: TimeSlot[]
  weekdayShortName: (dayIndex: number) => string
  emptyTitleText?: string
}

export function escapeHtml(text: string): string {
  const div = document.createElement("div")
  div.textContent = text
  return div.innerHTML
}

export function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

// 把 "HH:MM" 转为可比较的毫秒数，用于按时分先后排序。
export function parseTime(time: string): number {
  const [h, m] = (time || "").split(":").map(Number)
  return new Date(1970, 0, 1, h || 0, m || 0).getTime()
}

// 带圈数字序号（用于同时间段多首歌时的前缀），1-20 用 Unicode 带圈数字，超出回退到普通数字。
export function toCircledNumber(num: number): string {
  if (num >= 1 && num <= 10) {
    return String.fromCharCode(0x245f + num) // ①-⑩
  }
  if (num >= 11 && num <= 20) {
    return String.fromCharCode(0x2469 + (num - 10)) // ⑪-⑳
  }
  return `${num}`
}

// 对中文标题进行分词，在词组之间插入零宽空格，使导出图片按词组断行。
export function segmentChineseTitle(title: string): string {
  if (!title) return title
  try {
    const segs = segment.doSegment(title)
    return segs.map((s: any) => (s && typeof s === "object" ? s.w : String(s))).join("​")
  } catch {
    return title
  }
}

// 导出标题格式化：中文分词 + HTML 转义 + 支持 \n 换行。
export function formatExportTitle(title: string): string {
  if (!title) return ""
  const segmented = segmentChineseTitle(title)
  const escaped = escapeHtml(segmented)
  return escaped.replace(/\\n/g, "<br>")
}

function slotsForDate(dateStr: string, timeSlots: TimeSlot[]): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return timeSlots.filter(s => s.dayIndex === dayIdx).sort((a, b) => a.order - b.order)
}

function isMultiWeek(dates: string[]): boolean {
  return new Set(dates.map(d => dayjs(d).isoWeek())).size > 1
}

function dateCellHtml(date: string, weekdayShortName: (dayIndex: number) => string): string {
  const shortWd = escapeHtml(`周${weekdayShortName(dayIndexFromDate(date))}`)
  const formattedDate = escapeHtml(dayjs(date).format("MM.DD"))
  return `<div class="ex-simple-date-cell">
    <div class="ex-simple-date-month">${formattedDate}</div>
    <div class="ex-simple-date-weekday">${shortWd}</div>
  </div>`
}

export function buildDormTableHTML(dates: string[], options: BuildDormTableOptions): string {
  const { title = null, songs, timeSlots, weekdayShortName } = options
  const timeSet = new Set<string>()
  for (const date of dates) {
    for (const slot of slotsForDate(date, timeSlots)) {
      timeSet.add(slot.time)
    }
  }
  const times = Array.from(timeSet).sort((a, b) => parseTime(a) - parseTime(b))
  if (times.length === 0) return ""

  const firstColWidth = 15
  const otherWidth = Math.floor((100 - firstColWidth) / times.length * 100) / 100

  const rows = dates.map(date => {
    const daySlots = slotsForDate(date, timeSlots)
    const slotMap: Record<string, Song[]> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = songs.filter(s => s.date === date && s.timeSlotId === slot.id)
    }
    const cells = times.map(time => {
      const slotSongs = (slotMap[time] || [])
        .slice()
        .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
      const titleText = slotSongs
        .map((song, i) => {
          const title = song.title && song.title.trim() ? song.title : (options.emptyTitleText ?? "（未命名）")
          const titleHtml = formatExportTitle(title)
          if (slotSongs.length > 1) {
            return `<span class="song-with-number"><span class="circled-number">${toCircledNumber(i + 1)}</span><span class="song-title">${titleHtml}</span></span>`
          }
          return `<span class="song-with-number"><span class="song-title">${titleHtml}</span></span>`
        })
        .join("<br/>")
      return `<td>${titleText}</td>`
    }).join("")
    return `<tr><td class="ex-simple-table-td-center">${dateCellHtml(date, weekdayShortName)}</td>${cells}</tr>`
  }).join("")

  const headerCells = times.map(time => `<th>${escapeHtml(time)}</th>`).join("")

  const titleHtml = title ? `<h2 class="ex-simple-title">${escapeHtml(title)}</h2>` : ""

  return `${titleHtml}
    <table class="ex-simple-table">
      <colgroup>
        <col style="width:${firstColWidth}%;" />
        ${times.map(() => `<col style="width:${otherWidth}%;" />`).join("")}
      </colgroup>
      <thead>
        <tr><th></th>${headerCells}</tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>`
}
