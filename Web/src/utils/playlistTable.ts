import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import type { Song, TimeSlot } from "../api/types"

dayjs.extend(isoWeek)

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

// 罗马数字序号（用于同时间段多首歌时的前缀）。
export function toRoman(num: number): string {
  const map: [number, string][] = [
    [1000, "M"], [900, "CM"], [500, "D"], [400, "CD"],
    [100, "C"], [90, "XC"], [50, "L"], [40, "XL"],
    [10, "X"], [9, "IX"], [5, "V"], [4, "IV"], [1, "I"],
  ]
  let n = num
  let out = ""
  for (const [v, s] of map) {
    while (n >= v) {
      out += s
      n -= v
    }
  }
  return out || "I"
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
          const label = slotSongs.length > 1 ? `${toRoman(i + 1)}. ${title}` : title
          return escapeHtml(label)
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
