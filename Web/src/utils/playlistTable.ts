import type { Song, TimeSlot } from "../api/types"
import { dayjs, dayIndexFromDate, parseTime } from "./datetime"

// segmentit 自带的中文词典有 260 万字符（gzip 后约 1.25 MB），且构造 Segment 会
// 同步解析全部词条。静态 import 会把它焊进 /export 与 /organize 的路由块，
// 页面必须先下载完整个词典才能渲染。
//
// 因此改为动态 import：词典成为独立块，按需拉取。
// segmentChineseTitle 保持同步（调用方在 computed 里同步用它），词典未就绪时
// 原样返回标题——只是少了按词断行的零宽空格，不影响正确性。
// 需要分词生效的调用方应先 await ensureSegmenter()。
type Segmenter = { doSegment: (text: string) => unknown[] }

let segmenter: Segmenter | null = null
let loading: Promise<void> | null = null

/** 预加载中文分词词典。重复调用共享同一次加载。 */
export function ensureSegmenter(): Promise<void> {
  if (segmenter) return Promise.resolve()
  if (!loading) {
    loading = import("segmentit")
      .then((m) => {
        segmenter = m.useDefault(new m.Segment()) as Segmenter
      })
      .catch((err) => {
        // 加载失败不该让导出流程崩掉，退化成不分词。
        console.warn("加载中文分词词典失败，标题将不做分词断行:", err)
        loading = null
      })
  }
  return loading
}

/** 词典是否已就绪。供 UI 在加载完成后触发重新渲染。 */
export function isSegmenterReady(): boolean {
  return segmenter !== null
}

// dayIndexFromDate / parseTime 从 ./datetime 转出，保持既有 import 路径不变。
export { dayIndexFromDate, parseTime }

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
// 词典未加载时原样返回（见 ensureSegmenter）。
export function segmentChineseTitle(title: string): string {
  if (!title) return title
  if (!segmenter) return title
  try {
    const segs = segmenter.doSegment(title)
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
  const safeTimeSlots = timeSlots || []
  const safeSongs = songs || []
  const safeDates = dates || []
  const timeSet = new Set<string>()
  for (const date of safeDates) {
    for (const slot of slotsForDate(date, safeTimeSlots)) {
      timeSet.add(slot.time)
    }
  }
  const times = Array.from(timeSet).sort((a, b) => parseTime(a) - parseTime(b))
  if (times.length === 0) return ""

  const firstColWidth = 15
  const otherWidth = Math.floor((100 - firstColWidth) / times.length * 100) / 100

  const rows = safeDates.map(date => {
    const daySlots = slotsForDate(date, safeTimeSlots)
    const slotMap: Record<string, Song[]> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = safeSongs.filter(s => s.date === date && s.timeSlotId === slot.id)
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
