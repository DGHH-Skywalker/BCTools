import { computed } from "vue"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import { useAppConfig } from "./useAppConfig"
import { useI18n } from "../i18n"
import { buildDormTableHTML, parseTime, toCircledNumber, formatExportTitle, ensureSegmenter } from "../utils/playlistTable"
import { dayjs } from "../utils/datetime"
import { domToPng } from "modern-screenshot"
import DOMPurify from "dompurify"
import type { Song, SongType, TimeSlot } from "../api/types"

// 导出图片 DOM 的样式：之前抽到 Web/src/styles/export.css 里；该文件只为
// domToPng 注入样式使用，没有别的消费者——直接 inline 在这里，把「调用方
// 唯一」的 CSS 与逻辑合在一处。颜色 / 字体变量走 :root 上的 CSS Variables。
const exportStyles = `
/* Simple table export (dorm / broadcast) */
.ex-simple-title {
  text-align: center;
  margin-bottom: 24px;
  font-weight: 400;
}

.ex-simple-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-family: var(--font-body);
}

.ex-simple-table th,
.ex-simple-table td {
  padding: 8px;
  border: 1px solid var(--color-border-dark);
  text-align: center;
  font-weight: 400;
}

.ex-simple-table th {
  background: var(--color-bg-light);
}

.ex-simple-table td {
  word-break: keep-all;
  overflow-wrap: anywhere;
  line-break: strict;
}

.ex-simple-date-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  line-height: 1.2;
}

.ex-simple-date-month {
  font-family: var(--font-body);
  font-size: 18px;
}

.ex-simple-date-weekday {
  font-family: var(--font-slot);
  font-size: 24px;
}

/* Poster export */
.ex-poster {
  position: relative;
  width: 1080px;
  height: auto;
  padding: 120px 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  box-sizing: border-box;
  overflow: hidden;
  background: transparent;
}

.ex-poster-bg {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  transform: scale(1.08);
}

.ex-poster-bg-gradient {
  position: absolute;
  inset: 0;
}

.ex-poster-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.08);
}

.ex-poster-badge {
  position: absolute;
  top: 90px;
  right: 80px;
  display: flex;
  flex-direction: row;
  gap: 10px;
  z-index: 3;
}

.ex-poster-badge-text {
  font-family: var(--font-title);
  font-size: 34px;
  font-weight: 400;
  color: var(--theme-color);
  writing-mode: vertical-rl;
  text-orientation: upright;
  letter-spacing: 0.12em;
}

.ex-poster-title-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 28px;
  padding: 0 80px;
  box-sizing: border-box;
  z-index: 10;
}

.ex-poster-logo {
  height: 120px;
  width: auto;
  object-fit: contain;
  flex-shrink: 0;
  pointer-events: none;
  -webkit-user-drag: none;
  filter: drop-shadow(0 0.08em 0.15em rgba(0, 0, 0, 0.35));
}

.ex-poster-logo-placeholder {
  width: 54px;
  height: 105px;
  flex-shrink: 0;
}

.ex-poster-title {
  font-family: var(--font-title);
  font-size: 90px;
  font-weight: 400;
  color: var(--color-poster-text);
  text-shadow: 0 0.08em 0.15em rgba(0, 0, 0, 0.35);
  white-space: nowrap;
}

.ex-poster-card {
  position: relative;
  z-index: 1;
  width: 960px;
  background: rgba(0, 0, 0, 0.32);
  border-radius: var(--radius-2xl);
  box-shadow: 0 12px 38px 0 rgba(0, 0, 0, 0.45);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.ex-poster-card-header,
.ex-poster-card-footer {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-sizing: border-box;
}

.ex-poster-card-header {
  padding-top: 36px;
}

.ex-poster-card-footer {
  padding-bottom: 36px;
}

.ex-poster-quote {
  font-family: var(--font-fangsong);
  font-size: 36px;
  color: var(--color-poster-muted);
  text-align: center;
  letter-spacing: 0.05em;
  line-height: 1.4;
}

.ex-poster-divider {
  width: 90%;
  height: 1px;
  background: #ffffff;
  opacity: 0.6;
  margin: 24px 0 0;
}

.ex-poster-divider-bottom {
  margin: 0 0 24px;
}

.ex-poster-table-wrap {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 24px 0;
  box-sizing: border-box;
}

.ex-poster-empty {
  width: 94%;
  padding: 48px 0;
  text-align: center;
  color: var(--color-poster-muted);
  font-size: 36px;
  font-family: var(--font-body);
}

.ex-poster-grid {
  width: 94%;
  display: grid;
  gap: 0;
  align-items: stretch;
  justify-items: stretch;
  font-family: var(--font-body);
  border: 1px solid #ffffff;
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.ex-poster-grid.broadcast {
  border-color: #ffffff;
}

.ex-poster-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
  font-size: 36px;
  font-weight: 400;
  color: var(--color-poster-text);
  opacity: 1;
  background: transparent;
  padding: 0 12px;
  line-height: 1.25;
  box-sizing: border-box;
  min-height: 100%;
  word-break: keep-all;
  overflow-wrap: anywhere;
  line-break: strict;
}

.ex-poster-cell.narrow {
  padding: 0 8px;
  font-family: var(--font-body);
  font-size: 24px;
  font-weight: 100;
  opacity: 0.85;
}

.ex-poster-cell.header {
  font-size: 40px;
}

.ex-poster-cell.title {
  font-family: var(--font-song);
}

.ex-poster-cell.slot,
.ex-poster-cell.header {
  font-family: var(--font-slot);
}

.ex-poster-cell-stack {
  display: flex;
  flex-direction: column;
  gap: 6px;
  justify-content: center;
  width: 100%;
}

.ex-poster-cell-stack.loose {
  gap: 8px;
}

.ex-poster-afternoon-label {
  font-family: var(--font-title);
  font-size: 42px;
  color: var(--color-poster-text);
  font-weight: 400;
  line-height: 1.2;
  letter-spacing: 0.02em;
}

.song-with-number {
  display: inline-flex;
  align-items: baseline;
  justify-content: center;
  gap: 4px;
  line-height: 1.3;
}

.circled-number {
  font-size: 0.85em;
  opacity: 0.85;
  flex-shrink: 0;
}

.song-title {
  word-break: keep-all;
  overflow-wrap: anywhere;
  line-break: strict;
}
`


const fontBase64Cache: Record<string, string | null> = {}
const imageBase64Cache: Record<string, string | null> = {}

export function useExportImage() {
  const { t, weekdayShortName } = useI18n()
  const { config } = useAppConfig()
  const themeColor = computed(() => config.value.themeColor)
  const songsStore = useSongsStore()
  const settingsStore = useSettingsStore()
  const exportStore = useExportStore()

  function dayIndexFromDate(dateStr: string): number {
    const d = dayjs(dateStr)
    return d.day() === 0 ? 7 : d.day()
  }

  function slotsForDate(dateStr: string): TimeSlot[] {
    const dayIdx = dayIndexFromDate(dateStr)
    return (settingsStore.timeSlots || [])
      .filter((s) => s.dayIndex === dayIdx)
      .sort((a, b) => parseTime(a.time) - parseTime(b.time))
  }

  function escapeHtml(text: string): string {
    const div = document.createElement("div")
    div.textContent = text
    return div.innerHTML
  }

  // 空标题占位：dorm/broadcast 均允许空标题，导出时空标题留空（不显示占位符）
  function displayTitle(song: Song): string {
    return song.title && song.title.trim() ? song.title : ""
  }

  function songWithNumberHtml(title: string, index: number, showNumber: boolean): string {
    const titleHtml = formatExportTitle(title)
    if (!showNumber) return `<div class="song-with-number"><span class="song-title">${titleHtml}</span></div>`
    return `<div class="song-with-number"><span class="circled-number">${toCircledNumber(index + 1)}</span><span class="song-title">${titleHtml}</span></div>`
  }

  // 推断广播时段：优先用 period，缺失时用 createdAt（兜底 date）的小时，异常归下午
  function periodOf(s: Song): "noon" | "afternoon" {
    if (s.period === "noon" || s.period === "afternoon") return s.period
    const ts = s.createdAt || s.date
    if (ts) {
      const h = dayjs(ts).hour()
      if (!Number.isNaN(h)) return h < 12 ? "noon" : "afternoon"
    }
    return "afternoon"
  }

  function isMultiWeek(dates: string[]): boolean {
    return new Set(dates.map((d) => dayjs(d).isoWeek())).size > 1
  }

  function dateCellHtml(date: string): string {
    const shortWd = escapeHtml(`周${weekdayShortName(dayIndexFromDate(date))}`)
    const formattedDate = escapeHtml(dayjs(date).format("MM.DD"))
    return `<div class="ex-simple-date-cell">
      <div class="ex-simple-date-month">${formattedDate}</div>
      <div class="ex-simple-date-weekday">${shortWd}</div>
    </div>`
  }

  function getExportWeek(dates: string[]): number {
    if (dates.length === 0) return dayjs().isoWeek()
    const earliest = [...dates].sort()[0]
    return dayjs(earliest).isoWeek()
  }

  async function loadFontBase64(fontFile: string): Promise<string | null> {
    if (fontBase64Cache[fontFile] !== undefined) return fontBase64Cache[fontFile]
    try {
      const res = await fetch(`/fonts/${fontFile}`, { cache: "force-cache" })
      if (!res.ok) {
        fontBase64Cache[fontFile] = null
        return null
      }
      const buf = await res.arrayBuffer()
      const bytes = new Uint8Array(buf)
      let binary = ""
      const len = bytes.byteLength
      for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i])
      }
      fontBase64Cache[fontFile] = btoa(binary)
      return fontBase64Cache[fontFile]
    } catch {
      fontBase64Cache[fontFile] = null
      return null
    }
  }

  async function loadImageBase64(path: string): Promise<string | null> {
    if (imageBase64Cache[path] !== undefined) return imageBase64Cache[path]
    try {
      const res = await fetch(path, { cache: "force-cache" })
      if (!res.ok) {
        imageBase64Cache[path] = null
        return null
      }
      const blob = await res.blob()
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onloadend = () => resolve(reader.result as string)
        reader.onerror = reject
        reader.readAsDataURL(blob)
      })
      imageBase64Cache[path] = dataUrl
      return dataUrl
    } catch {
      imageBase64Cache[path] = null
      return null
    }
  }

  function buildBroadcastTable(dates: string[]): string {
    const allSongs: Song[] = []
    for (const date of dates) {
      allSongs.push(...songsStore.broadcastSongs.filter((s) => s.date === date))
    }
    allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)

    const rows = allSongs
      .map((s) => {
        const periodLabel = s.period === "noon" ? t("broadcast.noon") : t("broadcast.afternoon")
        return `<tr>
          <td class="ex-simple-table-td-center">${dateCellHtml(s.date)}</td>
          <td class="ex-simple-table-td-center">${escapeHtml(periodLabel)}</td>
          <td>${formatExportTitle(displayTitle(s))}</td>
          <td>${escapeHtml(s.remark)}</td>
        </tr>`
      })
      .join("")

    return `<h2 class="ex-simple-title">${escapeHtml(t("export.broadcast"))}</h2>
      <table class="ex-simple-table">
        <colgroup>
          <col style="width:18%;" />
          <col style="width:15%;" />
          <col style="width:42%;" />
          <col style="width:25%;" />
        </colgroup>
        <thead>
          <tr>
            <th></th>
            <th>时段</th>
            <th>歌名</th>
            <th>备注</th>
          </tr>
        </thead>
        <tbody>${rows}</tbody>
      </table>`
  }

  async function preloadImage(src: string): Promise<void> {
    return new Promise((resolve) => {
      const img = new Image()
      img.onload = () => resolve()
      img.onerror = () => resolve()
      img.src = src
    })
  }

  async function buildPosterHTML(dates: string[], type: SongType): Promise<string> {
    const titleText = escapeHtml(type === "dorm" ? t("export.dorm") : t("export.broadcast"))
    const badge = exportStore.vacationBadge
    const blurPx = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
    const quoteValue = type === "dorm" ? exportStore.posterQuoteDorm : exportStore.posterQuoteBroadcast
    const quoteText = escapeHtml(quoteValue || "人生南北多歧路 君向潇湘我向秦")

    const badgeHtml =
      badge === "none"
        ? ""
        : `<div class="ex-poster-badge">
            <div class="ex-poster-badge-text">${badge === "summer" ? "暑假" : "寒假"}</div>
            <div class="ex-poster-badge-text">限定</div>
          </div>`

    const bgImage = exportStore.backgroundImageFor(type)
    if (bgImage) {
      await preloadImage(bgImage)
    }
    const bgLayer = bgImage
      ? `<div class="ex-poster-bg" style="background-image:url(${bgImage});filter:blur(${blurPx}px) brightness(0.95);"></div>`
      : `<div class="ex-poster-bg ex-poster-bg-gradient" style="background:linear-gradient(135deg,${themeColor.value} 0%,#004d70 50%,#1a1a1a 100%);"></div>`

    const tableHtml = type === "dorm" ? buildDormPosterTable(dates) : buildBroadcastPosterTableV2(dates)

    const logoSrc = await loadImageBase64("/logo-white.png")
    const logoHtml = logoSrc
      ? `<img class="ex-poster-logo" src="${logoSrc}" alt="logo" />`
      : `<div class="ex-poster-logo-placeholder"></div>`

    const titleHtml = `<div class="ex-poster-title-wrap">
      ${logoHtml}
      <div class="ex-poster-title">${titleText}</div>
    </div>`

    return `<div style="position:absolute;inset:0;overflow:hidden;z-index:0;">
      ${bgLayer}
      <div class="ex-poster-overlay"></div>
    </div>
    ${badgeHtml}
    <div class="ex-poster">
      ${titleHtml}
      <div class="ex-poster-card">
        <div class="ex-poster-card-header">
          <div class="ex-poster-quote">${quoteText}</div>
          <div class="ex-poster-divider"></div>
        </div>
        <div class="ex-poster-table-wrap">${tableHtml}</div>
        <div class="ex-poster-card-footer">
          <div class="ex-poster-divider ex-poster-divider-bottom"></div>
          <div class="ex-poster-quote">&nbsp;</div>
        </div>
      </div>
    </div>`
  }

  function buildDormPosterTable(dates: string[]): string {
    const timeSet = new Set<string>()
    for (const date of dates) {
      for (const slot of slotsForDate(date)) {
        timeSet.add(slot.time)
      }
    }
    const times = Array.from(timeSet).sort((a, b) => parseTime(a) - parseTime(b))
    const type: SongType = "dorm"

    if (times.length === 0) {
      return `<div class="ex-poster-empty">${escapeHtml(t("export.noSlots"))}</div>`
    }

    const colCount = times.length + 1
    const headerCells = [
      cellHtml("", true, false, 0, colCount, 0, "slot", type),
      ...times.map((time, i) => cellHtml(time, true, false, i + 1, colCount, 0, "slot", type)),
    ].join("")
    const headerRow = rowHtml(headerCells, true, 0)

    const dataRows = dates
      .map((date, rowIdx) => {
        const daySlots = slotsForDate(date)
        const slotMap: Record<string, Song[]> = {}
        for (const slot of daySlots) {
          slotMap[slot.time] = songsStore.dormSongs.filter((s) => s.date === date && s.timeSlotId === slot.id)
        }
        const cells = [
          cellHtml(dateCellHtml(date), false, true, 0, colCount, rowIdx + 1, "slot", type),
          ...times.map((time, i) => {
            const slotSongs = (slotMap[time] || [])
              .slice()
              .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
            const inner = slotSongs.length > 0
              ? `<div class="ex-poster-cell-stack">${slotSongs
                  .map((s, idx) => songWithNumberHtml(displayTitle(s), idx, slotSongs.length > 1))
                  .join("")}</div>`
              : ""
            return cellHtml(inner, false, false, i + 1, colCount, rowIdx + 1, "title", type)
          }),
        ].join("")
        return rowHtml(cells, false, rowIdx + 1)
      })
      .join("")

    return gridWrapperHtml(headerRow + dataRows, colCount, dates.length, true, type)
  }

  function buildBroadcastPosterTable(dates: string[]): string {
    const allSongs: Song[] = []
    for (const date of dates) {
      allSongs.push(...songsStore.broadcastSongs.filter((s) => s.date === date))
    }
    allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)
    const type: SongType = "broadcast"

    const headers = ["", "歌名", "备注"]
    const colCount = headers.length
    const headerCells = headers
      .map((h, i) => {
        const fontRole = i === 0 ? "slot" : undefined
        return cellHtml(h, true, i === 0, i, colCount, 0, fontRole, type)
      })
      .join("")
    const headerRow = rowHtml(headerCells, true, 0)

    const dataRows = allSongs
      .map((s, rowIdx) => {
        const cells = [
          cellHtml(dateCellHtml(s.date), false, true, 0, colCount, rowIdx + 1, "slot", type),
          cellHtml(formatExportTitle(displayTitle(s)), false, false, 1, colCount, rowIdx + 1, "title", type),
          cellHtml(escapeHtml(s.remark), false, false, 2, colCount, rowIdx + 1, undefined, type),
        ].join("")
        return rowHtml(cells, false, rowIdx + 1)
      })
      .join("")

    return gridWrapperHtml(headerRow + dataRows, colCount, allSongs.length, false, type)
  }

  // 下午单元格首行的栏目标签：取当日 broadcastColumnMap 的栏目名，用白色江西拙楷、
  // 略大于歌名的字号，作为当天下午栏目的固定标识。
  function afternoonLabelHtml(date: string): string {
    const colName = settingsStore.broadcastColumnMap[String(dayIndexFromDate(date))] || ""
    if (!colName || colName === "无") return ""
    return `<div class="ex-poster-afternoon-label">${escapeHtml(colName)}</div>`
  }

  function buildBroadcastPosterTableV2(dates: string[]): string {
    // 广播歌单若按「每天一列」展示，一周有 7 天时每一列只剩约 110px，
    // 中文歌名会被逐字折行，继而把整张海报异常拉长。改为「每天一行、
    // 中午/下午两列」，无论选择多少天，歌名列都保有稳定的可读宽度。
    const colCount = 3
    const type: SongType = "broadcast"

    const headerCells = ["", "中午", "下午"]
      .map((label, colIdx) => cellHtml(label, true, colIdx === 0, colIdx, colCount, 0, "slot", type))
      .join("")
    const headerRow = rowHtml(headerCells, true, 0)

    const dataRows = dates
      .map((date, rowIdx) => {
        const songs = songsStore.broadcastSongs.filter((s) => s.date === date)
        const cells = [
          cellHtml(dateCellHtml(date), false, true, 0, colCount, rowIdx + 1, "slot", type),
          ...(["noon", "afternoon"] as const).map((period, colIdx) => {
            const targetSongs = songs.filter((s) => periodOf(s) === period)
            const parts: string[] = []
            if (period === "afternoon") {
              const lbl = afternoonLabelHtml(date)
              if (lbl) parts.push(lbl)
            }
            const sortedSongs = targetSongs
              .slice()
              .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
            sortedSongs.forEach((s, idx) => {
              parts.push(songWithNumberHtml(displayTitle(s), idx, sortedSongs.length > 1))
            })
            const inner = parts.length > 0
              ? `<div class="ex-poster-cell-stack loose">${parts.join("")}</div>`
              : ""
            return cellHtml(inner, false, false, colIdx + 1, colCount, rowIdx + 1, "title", type)
          }),
        ].join("")
        return rowHtml(cells, false, rowIdx + 1)
      })
      .join("")

    return gridWrapperHtml(headerRow + dataRows, colCount, dates.length, true, type)
  }

  function gridWrapperHtml(
    rowsHtml: string,
    colCount: number,
    rowCount: number,
    hasDateCol: boolean = false,
    type: SongType = "dorm"
  ): string {
    const cols = hasDateCol ? `140px repeat(${colCount - 1}, 1fr)` : `repeat(${colCount}, 1fr)`
    // 导出图片的分割线统一固定白色，**不允许**跟随主题色：
    // 海报底色是 #1a1a1a，跟随主题色在用户改主题后会出现对比度不足。
    const borderColor = "#ffffff"
    return `<div class="ex-poster-grid ${type}" style="grid-template-columns:${cols};grid-template-rows:auto repeat(${rowCount},minmax(100px,auto));border-color:${borderColor};">
      ${rowsHtml}
    </div>`
  }

  function rowHtml(cellsHtml: string, _isHeader?: boolean, _rowIndex?: number): string {
    return cellsHtml
  }

  function cellHtml(
    content: string,
    isHeader: boolean,
    narrow: boolean,
    colIndex: number,
    totalCols: number,
    rowIndex: number,
    fontRole?: "title" | "slot",
    type: SongType = "dorm"
  ): string {
    const classes = ["ex-poster-cell"]
    if (isHeader) classes.push("header")
    if (narrow) classes.push("narrow")
    if (fontRole) classes.push(fontRole)

    // 单元格分割线也是固定白色，理由同 gridWrapperHtml。
    const borderColor = "#ffffff"
    const borderBottom = type === "broadcast" ? "" : `border-bottom:1px solid ${borderColor};`
    const borderRight = type === "dorm" || colIndex === totalCols - 1 ? "" : `border-right:1px solid ${borderColor};`

    return `<div class="${classes.join(" ")}" style="${borderBottom}${borderRight}">${content}</div>`
  }

  async function generateImage(dates: string[], type: SongType): Promise<string> {
    // 导出是唯一真正需要中文分词的场景，在这里等词典就绪。
    // 词典是动态 import 的独立块（约 1.25 MB gzip），不进首屏。
    await ensureSegmenter()
    const isPoster = !exportStore.simpleMode
    let rawHtml = ""
    let bgColor = "#ffffff"
    if (isPoster) {
      rawHtml = await buildPosterHTML(dates, type)
      bgColor = "#1a1a1a"
    } else {
      rawHtml =
        type === "dorm"
          ? buildDormTableHTML(dates, {
              title: t("export.dorm"),
              songs: songsStore.dormSongs || [],
              timeSlots: settingsStore.timeSlots || [],
              weekdayShortName,
              emptyTitleText: "",
            })
          : buildBroadcastTable(dates)
    }

    if (!rawHtml || !rawHtml.trim()) {
      throw new Error(t("export.noData"))
    }

    const wrapper = document.createElement("div")
    wrapper.style.cssText =
      "position:fixed;left:0;top:0;width:0;height:0;overflow:hidden;pointer-events:none;z-index:-1;"
    const container = document.createElement("div")
    container.style.cssText = isPoster
      ? "position:absolute;left:0;top:0;width:1080px;height:auto;background:" + bgColor + ";"
      : "position:absolute;left:0;top:0;width:1080px;padding:48px;background:#fff;font-family:sans-serif;height:auto;"
    container.innerHTML = DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: [
        "div",
        "span",
        "svg",
        "path",
        "rect",
        "circle",
        "polygon",
        "style",
        "h2",
        "table",
        "thead",
        "tbody",
        "tr",
        "td",
        "th",
        "img",
        "br",
      ],
      ALLOWED_ATTR: [
        "style",
        "class",
        "viewBox",
        "fill",
        "stroke",
        "stroke-width",
        "cx",
        "cy",
        "r",
        "x",
        "y",
        "width",
        "height",
        "points",
        "rx",
        "d",
        "xmlns",
        "src",
        "alt",
      ],
    })

    const fontFiles = ["江西拙楷3.0.ttf", "方正颜宋简体.ttf", "方正小标宋简.TTF", "仿宋_GB2312.ttf"]
    const fontFamilies = ["JiangxiZhuokai", "FZYanSong", "FZXiaoBiaoSong", "FangSong_GB2312"]
    const style = document.createElement("style")
    let styleContent = exportStyles
    for (let i = 0; i < fontFiles.length; i++) {
      const base64 = await loadFontBase64(fontFiles[i])
      if (base64) {
        styleContent += `@font-face{font-family:'${fontFamilies[i]}';src:url('data:font/truetype;base64,${base64}') format('truetype');font-weight:normal;font-style:normal;}`
      }
    }
    if (styleContent) {
      style.textContent = styleContent
      container.appendChild(style)
      for (const family of fontFamilies) {
        try {
          await Promise.race([document.fonts.load(`12px ${family}`), new Promise((resolve) => setTimeout(resolve, 3000))])
        } catch {
          // ignore
        }
      }
    }

    wrapper.appendChild(container)
    document.body.appendChild(wrapper)

    try {
      return await domToPng(container, {
        scale: type === "broadcast" ? 3 : 2,
        backgroundColor: bgColor,
      })
    } catch (err) {
      console.error("导出图片失败:", err)
      throw err
    } finally {
      document.body.removeChild(wrapper)
    }
  }

  return {
    generateImage,
    getExportWeek,
  }
}
