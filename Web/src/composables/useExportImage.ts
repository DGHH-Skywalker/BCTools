import { computed } from "vue"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import { useAppConfig } from "./useAppConfig"
import { useI18n } from "../i18n"
import { buildDormTableHTML, parseTime, toCircledNumber, formatExportTitle } from "../utils/playlistTable"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import { domToPng } from "modern-screenshot"
import DOMPurify from "dompurify"
import type { Song, SongType, TimeSlot } from "../api/types"
import exportStyles from "../styles/export.css?inline"

dayjs.extend(isoWeek)

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
    const rowLabels = ["中午", "下午"]
    const colCount = dates.length + 1
    const type: SongType = "broadcast"

    const headerCells = [
      cellHtml("", true, false, 0, colCount, 0, "slot", type),
      ...dates.map((date, i) => cellHtml(dateCellHtml(date), true, false, i + 1, colCount, 0, "slot", type)),
    ].join("")
    const headerRow = rowHtml(headerCells, true, 0)

    const dataRows = rowLabels
      .map((label, rowIdx) => {
        const cells = [
          cellHtml(label, false, true, 0, colCount, rowIdx + 1, "slot", type),
          ...dates.map((date, colIdx) => {
            const songs = songsStore.broadcastSongs.filter((s) => s.date === date)
            const targetSongs = songs.filter((s) => periodOf(s) === (rowIdx === 0 ? "noon" : "afternoon"))
            const parts: string[] = []
            if (rowIdx === 1) {
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

    return gridWrapperHtml(headerRow + dataRows, colCount, 2, true, type)
  }

  function gridWrapperHtml(
    rowsHtml: string,
    colCount: number,
    rowCount: number,
    hasDateCol: boolean = false,
    type: SongType = "dorm"
  ): string {
    const cols = hasDateCol ? `140px repeat(${colCount - 1}, 1fr)` : `repeat(${colCount}, 1fr)`
    const borderColor = type === "broadcast" ? "var(--color-poster-table-border-broadcast)" : "var(--color-poster-table-border)"
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

    const borderColor = type === "broadcast" ? "var(--color-poster-table-border-broadcast)" : "var(--color-poster-table-border)"
    const borderBottom = type === "broadcast" ? "" : `border-bottom:1px solid ${borderColor};`
    const borderRight = type === "dorm" || colIndex === totalCols - 1 ? "" : `border-right:1px solid ${borderColor};`

    return `<div class="${classes.join(" ")}" style="${borderBottom}${borderRight}">${content}</div>`
  }

  async function generateImage(dates: string[], type: SongType): Promise<string> {
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
