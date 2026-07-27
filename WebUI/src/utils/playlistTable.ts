import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import type { Song, TimeSlot } from "../api/types"

dayjs.extend(isoWeek)

export interface BuildDormTableOptions {
  title?: string | null
  songs: Song[]
  timeSlots: TimeSlot[]
  weekdayShortName: (dayIndex: number) => string
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
  return `<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;line-height:1.2;">
    <div style="font-family:'HarmonyOS Sans SC','HarmonyOS Sans','DengXian','Microsoft YaHei','PingFang SC',sans-serif;font-size:18px;">${formattedDate}</div>
    <div style="font-family:'FZXiaoBiaoSong','方正小标宋简','STSong','SimSun','Songti SC',serif;font-size:24px;">${shortWd}</div>
  </div>`
}

export function buildDormTableHTML(dates: string[], options: BuildDormTableOptions): string {
  const { title = null, songs, timeSlots, weekdayShortName } = options
  const timeOrder: Record<string, number> = {}
  for (const date of dates) {
    for (const slot of slotsForDate(date, timeSlots)) {
      if (timeOrder[slot.time] === undefined) timeOrder[slot.time] = slot.order
    }
  }
  const times = Object.keys(timeOrder).sort((a, b) => timeOrder[a] - timeOrder[b])
  if (times.length === 0) return ""

  const firstColWidth = 15
  const otherWidth = Math.floor((100 - firstColWidth) / times.length * 100) / 100

  const rows = dates.map(date => {
    const daySlots = slotsForDate(date, timeSlots)
    const slotMap: Record<string, Song | undefined> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = songs.find(s => s.date === date && s.timeSlotId === slot.id)
    }
    const cells = times.map(time => {
      const song = slotMap[time]
      return `<td style="padding:8px;border:1px solid #ddd;text-align:center;font-weight:400;word-break:break-word;">${song ? escapeHtml(song.title) : ""}</td>`
    }).join("")
    return `<tr><td style="padding:8px;border:1px solid #ddd;text-align:center;white-space:nowrap;font-weight:400;">${dateCellHtml(date, weekdayShortName)}</td>${cells}</tr>`
  }).join("")

  const headerCells = times.map(time => `<th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;font-weight:400;">${escapeHtml(time)}</th>`).join("")

  const titleHtml = title ? `<h2 style="text-align:center;margin-bottom:24px;font-weight:400;">${escapeHtml(title)}</h2>` : ""

  return `${titleHtml}
    <table style="width:100%;border-collapse:collapse;table-layout:fixed;">
      <colgroup><col style="width:${firstColWidth}%;" />${times.map(() => `<col style="width:${otherWidth}%;" />`).join("")}</colgroup>
      <thead><tr><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;font-weight:400;"></th>${headerCells}</tr></thead>
      <tbody>${rows}</tbody>
    </table>`
}
