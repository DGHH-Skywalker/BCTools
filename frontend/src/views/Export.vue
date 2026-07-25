<template>
  <div style="padding:16px;">
    <n-h2>{{ t("export.title") }}</n-h2>
    <n-radio-group v-model:value="songType" style="margin-bottom:12px;">
      <n-radio value="dorm">{{ t("export.dorm") }}</n-radio>
      <n-radio value="broadcast">{{ t("export.broadcast") }}</n-radio>
    </n-radio-group>
    <n-space style="margin-bottom:12px;">
      <n-button size="small" @click="quickSelect(0)">{{ t("export.thisWeek") }}</n-button>
      <n-button size="small" @click="quickSelect(-1)">{{ t("export.lastWeek") }}</n-button>
      <n-button size="small" @click="quickSelect(1)">{{ t("export.nextWeek") }}</n-button>
    </n-space>
    <n-radio-group v-model:value="exportStore.templateType" style="margin-bottom:12px;display:block;">
      <n-radio value="simple">{{ t("export.templateSimple") }}</n-radio>
      <n-radio value="poster">{{ t("export.templatePoster") }}</n-radio>
    </n-radio-group>
    <div v-if="exportStore.templateType === 'poster'" style="margin-bottom:12px;padding:12px;border:1px solid #e0e0e0;border-radius:6px;">
      <n-space vertical>
        <n-space align="center">
          <span>{{ t("export.background") }}</span>
          <input type="file" accept="image/*" @change="onBackgroundFileChange">
          <n-button v-if="exportStore.backgroundImage" size="tiny" @click="exportStore.backgroundImage = null">{{ t("common.delete") }}</n-button>
        </n-space>
        <n-space align="center">
          <span>{{ t("export.backgroundColor") }}</span>
          <n-color-picker v-model:value="exportStore.backgroundColor" style="width:160px;" :show-alpha="false" />
        </n-space>
      </n-space>
    </div>
    <CalendarGrid :year="year" :month="month" :songs-map="songsMap" selection-mode="multiple" :selected-dates="exportStore.selectedDates" @selection-change="exportStore.setDates" @update:year="year = $event" @update:month="month = $event" />
    <n-space wrap style="margin-top:8px;">
      <n-tag v-for="d in exportStore.selectedDates" :key="d" closable @close="exportStore.removeDate(d)">{{ d }}</n-tag>
    </n-space>
    <n-space style="margin-top:16px;" v-if="exportStore.selectedDates.length > 0">
      <n-button type="primary" @click="exportPlaylist" :loading="exporting">
        {{ t("export.exportBatch", { count: exportStore.selectedDates.length }) }}
      </n-button>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed } from "vue"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import CalendarGrid from "../components/calendar/CalendarGrid.vue"
import dayjs from "dayjs"
import { useMessage } from "naive-ui"
import { domToPng } from "modern-screenshot"
import DOMPurify from "dompurify"
import type { Song, TimeSlot } from "../api/types"

const { t, weekdayName } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const exportStore = useExportStore()
const message = useMessage()
const songType = ref<"dorm" | "broadcast">("dorm")
const year = ref(dayjs().year())
const month = ref(dayjs().month() + 1)
const exporting = ref(false)

const songsMap = computed(() => {
  const map: Record<string, { totalSlots: number; filledSlots: number }> = {}
  const list = songType.value === "dorm" ? songsStore.dormSongs : songsStore.broadcastSongs
  for (const s of list) {
    if (!map[s.date]) {
      const dayIdx = dayIndexFromDate(s.date)
      const total = settingsStore.timeSlots.filter(slot => slot.dayIndex === dayIdx).length
      map[s.date] = { totalSlots: total, filledSlots: 0 }
    }
    map[s.date].filledSlots++
  }
  return map
})

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), songsStore.fetchSongs("broadcast"), settingsStore.fetchSettings()])
})

watch(songType, () => exportStore.setDates([]))

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function quickSelect(offset: number) {
  const d = dayjs().add(offset * 7, "day")
  const start = d.startOf("week").add(1, "day")
  const end = d.endOf("week").add(1, "day")
  const dates: string[] = []
  let cur = start
  while (cur.isBefore(end) || cur.isSame(end, "day")) { dates.push(cur.format("YYYY-MM-DD")); cur = cur.add(1, "day") }
  exportStore.setDates(dates)
}

function escapeHtml(text: string): string {
  const div = document.createElement("div")
  div.textContent = text
  return div.innerHTML
}

function slotsForDate(dateStr: string): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return settingsStore.timeSlots.filter(s => s.dayIndex === dayIdx).sort((a, b) => a.order - b.order)
}

function onBackgroundFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    exportStore.backgroundImage = reader.result as string
  }
  reader.onerror = () => {
    message.error("背景图读取失败")
  }
  reader.readAsDataURL(file)
}

async function exportPlaylist() {
  const dates = exportStore.selectedDates
  if (dates.length === 0) return
  exporting.value = true
  try {
    const dataUrl = await generateImage(dates)
    const a = document.createElement("a")
    a.href = dataUrl
    a.download = `playlist-${songType.value}.png`
    a.click()
    message.success(t("export.success"))
  } catch (err) {
    message.error("导出失败")
    console.error(err)
  } finally {
    exporting.value = false
  }
}

async function generateImage(dates: string[]): Promise<string> {
  const isMobile = navigator.maxTouchPoints > 1 || window.innerWidth < 768
  const pixelRatio = isMobile ? 1 : 2
  const div = document.createElement("div")

  let rawHtml = ""
  let bgColor = "#ffffff"
  if (exportStore.templateType === "poster") {
    rawHtml = buildPosterHTML(dates)
    bgColor = exportStore.backgroundColor || "#1a1a1a"
    div.style.cssText = `width:1000px;height:1250px;position:relative;overflow:hidden;background:${bgColor};`
  } else {
    div.style.cssText = "width:1000px;padding:32px;background:#fff;font-family:sans-serif;"
    rawHtml = songType.value === "dorm" ? buildDormTable(dates) : buildBroadcastTable(dates)
  }

  div.innerHTML = DOMPurify.sanitize(rawHtml, {
    ALLOWED_TAGS: exportStore.templateType === "poster"
      ? ["div", "span", "svg", "path", "rect", "circle", "polygon", "style", "h2", "table", "thead", "tbody", "tr", "td", "th"]
      : ["h2", "table", "thead", "tbody", "tr", "td", "th"],
    ALLOWED_ATTR: ["style", "class", "viewBox", "fill", "stroke", "stroke-width", "cx", "cy", "r", "x", "y", "width", "height", "points", "rx", "d", "xmlns"],
  })
  document.body.appendChild(div)
  try {
    return await domToPng(div, { scale: pixelRatio, backgroundColor: bgColor })
  } finally {
    document.body.removeChild(div)
  }
}

function buildDormTable(dates: string[]): string {
  const timeOrder: Record<string, number> = {}
  for (const date of dates) {
    for (const slot of slotsForDate(date)) {
      if (timeOrder[slot.time] === undefined) timeOrder[slot.time] = slot.order
    }
  }
  const times = Object.keys(timeOrder).sort((a, b) => timeOrder[a] - timeOrder[b])

  const rows = dates.map(date => {
    const daySlots = slotsForDate(date)
    const slotMap: Record<string, Song | undefined> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = songsStore.dormSongs.find(s => s.date === date && s.timeSlotId === slot.id)
    }
    const cells = times.map(time => {
      const song = slotMap[time]
      return `<td style="padding:8px;border:1px solid #ddd;text-align:center;">${song ? escapeHtml(song.title) : "-"}</td>`
    }).join("")
    return `<tr><td style="padding:8px;border:1px solid #ddd;text-align:center;white-space:nowrap;">${escapeHtml(date)}<br/>${escapeHtml(weekdayName(dayIndexFromDate(date)))}</td>${cells}</tr>`
  }).join("")

  const headerCells = times.map(time => `<th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">${escapeHtml(time)}</th>`).join("")

  return `<h2 style="text-align:center;margin-bottom:24px;">${escapeHtml(t("export.dorm"))}</h2>
    <table style="width:100%;border-collapse:collapse;">
      <thead><tr><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">日期</th>${headerCells}</tr></thead>
      <tbody>${rows}</tbody>
    </table>`
}

function buildBroadcastTable(dates: string[]): string {
  const allSongs: Song[] = []
  for (const date of dates) {
    allSongs.push(...songsStore.broadcastSongs.filter(s => s.date === date))
  }
  allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)

  const rows = allSongs.map(s =>
    `<tr><td style="padding:8px;border:1px solid #ddd;text-align:center;white-space:nowrap;">${escapeHtml(s.date)}</td>` +
    `<td style="padding:8px;border:1px solid #ddd;text-align:center;">${escapeHtml(weekdayName(dayIndexFromDate(s.date)))}</td>` +
    `<td style="padding:8px;border:1px solid #ddd;">${escapeHtml(s.title)}</td>` +
    `<td style="padding:8px;border:1px solid #ddd;">${escapeHtml(s.remark)}</td></tr>`
  ).join("")

  return `<h2 style="text-align:center;margin-bottom:24px;">${escapeHtml(t("export.broadcast"))}</h2>
    <table style="width:100%;border-collapse:collapse;">
      <thead><tr><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">日期</th><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">星期</th><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">歌名</th><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;">备注</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>`
}

function buildPosterHTML(dates: string[]): string {
  const titleText = escapeHtml(songType.value === "dorm" ? t("export.dorm") : t("export.broadcast"))
  const badgeText = "暑假限定"
  const quoteText = "「人生南北多歧路 君向潇湘我向秦」"

  const bgImage = exportStore.backgroundImage
  const bgLayer = bgImage
    ? `<div style="position:absolute;inset:0;background-image:url(${bgImage});background-size:cover;background-position:center;filter:blur(18px) brightness(0.75);transform:scale(1.08);"></div>`
    : `<div style="position:absolute;inset:0;background:linear-gradient(135deg,#4a1c12 0%,#1a1a1a 50%,#0d0d0d 100%);"></div>`

  const tableHtml = songType.value === "dorm"
    ? buildDormPosterTable(dates)
    : buildBroadcastPosterTable(dates)

  return `<div style="position:absolute;inset:0;overflow:hidden;">
    ${bgLayer}
    <div style="position:absolute;inset:0;background:rgba(0,0,0,0.18);"></div>
  </div>
  <div style="position:absolute;left:0;right:0;top:0;height:200px;display:flex;align-items:center;justify-content:center;gap:24px;padding:0 70px;box-sizing:border-box;">
    <div style="width:45px;height:87.5px;flex-shrink:0;">
      <svg viewBox="0 0 100 100" style="width:100%;height:100%;display:block;" xmlns="http://www.w3.org/2000/svg">
        <polygon points="50,5 65,25 35,25" fill="#ffffff"/>
        <rect x="35" y="25" width="30" height="55" rx="5" fill="#ffffff"/>
        <circle cx="50" cy="52.5" r="6" fill="#111111"/>
        <path d="M28 38 Q12 52.5 28 67" stroke="#ffffff" stroke-width="4" fill="none" stroke-linecap="round"/>
        <path d="M18 31 Q-2 52.5 18 74" stroke="#ffffff" stroke-width="4" fill="none" stroke-linecap="round"/>
        <path d="M8 24 Q-16 52.5 8 81" stroke="#ffffff" stroke-width="4" fill="none" stroke-linecap="round"/>
      </svg>
    </div>
    <div style="font-family:'STSong','SimSun','Songti SC',serif;font-size:78px;font-weight:900;color:#ffffff;text-shadow:0 0.08em 0.15em rgba(0,0,0,0.35);white-space:nowrap;">${titleText}</div>
    <div style="margin-left:auto;font-family:'Microsoft YaHei','PingFang SC','Hiragino Sans GB',sans-serif;font-size:28px;font-weight:600;color:#ffffff;writing-mode:vertical-rl;text-orientation:upright;letter-spacing:0.15em;">${badgeText}</div>
  </div>
  <div style="position:absolute;left:90px;top:137.5px;width:820px;height:925px;background:rgba(0,0,0,0.72);border-radius:40px;box-shadow:0 12px 38px 0 rgba(0,0,0,0.45);overflow:hidden;display:flex;flex-direction:column;align-items:center;">
    <div style="margin-top:28px;font-family:'STKaiti','KaiTi','STKaiti SC',serif;font-size:24px;color:#d7d7d7;text-align:center;letter-spacing:0.05em;">${quoteText}</div>
    <div style="width:92%;height:1px;background:rgba(215,215,215,0.25);margin:18px 0;"></div>
    ${tableHtml}
  </div>`
}

function buildDormPosterTable(dates: string[]): string {
  const timeOrder: Record<string, number> = {}
  for (const date of dates) {
    for (const slot of slotsForDate(date)) {
      if (timeOrder[slot.time] === undefined) timeOrder[slot.time] = slot.order
    }
  }
  const times = Object.keys(timeOrder).sort((a, b) => timeOrder[a] - timeOrder[b])

  const headerCells = [
    cellHtml("星期", true, true),
    ...times.map(time => cellHtml(time, true, false))
  ].join("")

  const headerRow = rowHtml(headerCells, true)

  const dataRows = dates.map(date => {
    const daySlots = slotsForDate(date)
    const slotMap: Record<string, Song | undefined> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = songsStore.dormSongs.find(s => s.date === date && s.timeSlotId === slot.id)
    }
    const cells = [
      cellHtml(`${escapeHtml(weekdayName(dayIndexFromDate(date)))}`, false, true),
      ...times.map(time => cellHtml(slotMap[time] ? escapeHtml(slotMap[time]!.title) : "-", false, false))
    ].join("")
    return rowHtml(cells, false)
  }).join("")

  return gridWrapperHtml(headerRow + dataRows, times.length + 1)
}

function buildBroadcastPosterTable(dates: string[]): string {
  const allSongs: Song[] = []
  for (const date of dates) {
    allSongs.push(...songsStore.broadcastSongs.filter(s => s.date === date))
  }
  allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)

  const headerCells = ["日期", "星期", "歌名", "备注"].map((h, i) => cellHtml(h, true, i === 0 || i === 1)).join("")
  const headerRow = rowHtml(headerCells, true)

  const dataRows = allSongs.map(s => {
    const cells = [
      cellHtml(escapeHtml(s.date), false, true),
      cellHtml(escapeHtml(weekdayName(dayIndexFromDate(s.date))), false, true),
      cellHtml(escapeHtml(s.title), false, false),
      cellHtml(escapeHtml(s.remark), false, false)
    ].join("")
    return rowHtml(cells, false)
  }).join("")

  return gridWrapperHtml(headerRow + dataRows, 4)
}

function gridWrapperHtml(rowsHtml: string, colCount: number): string {
  let columns = ""
  if (colCount === 5) {
    columns = "0.9fr repeat(4,2.25fr)"
  } else if (colCount === 4) {
    columns = "1.1fr 0.9fr 2.5fr 1.5fr"
  } else {
    columns = `repeat(${colCount},1fr)`
  }

  return `<div style="width:92%;height:820px;display:grid;grid-template-columns:${columns};grid-template-rows:9% repeat(auto-fill,13%);gap:1px;align-items:stretch;justify-items:stretch;background:rgba(215,215,215,0.65);font-family:'Microsoft YaHei','PingFang SC','Hiragino Sans GB',sans-serif;">
    ${rowsHtml}
  </div>`
}

function rowHtml(cellsHtml: string, isHeader: boolean): string {
  return cellsHtml
}

function cellHtml(content: string, isHeader: boolean, narrow: boolean): string {
  const fontFamily = isHeader
    ? "'STHeiti','SimHei','Heiti SC','Microsoft YaHei',sans-serif"
    : "'Microsoft YaHei','PingFang SC','Hiragino Sans GB',sans-serif"
  const fontSize = isHeader ? "22px" : "20px"
  const fontWeight = isHeader ? "700" : "600"
  const color = "#ffffff"
  const padding = narrow ? "0 4px" : "0 8px"
  return `<div style="display:flex;justify-content:center;align-items:center;text-align:center;font-family:${fontFamily};font-size:${fontSize};font-weight:${fontWeight};color:${color};background:transparent;padding:${padding};line-height:1.35;box-sizing:border-box;min-height:100%;word-break:break-word;">${content}</div>`
}
</script>
