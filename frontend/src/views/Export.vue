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
  div.style.cssText = "width:1000px;padding:32px;background:#fff;font-family:sans-serif;"

  let rawHtml = ""
  if (songType.value === "dorm") {
    rawHtml = buildDormTable(dates)
  } else {
    rawHtml = buildBroadcastTable(dates)
  }

  div.innerHTML = DOMPurify.sanitize(rawHtml, {
    ALLOWED_TAGS: ["h2", "table", "thead", "tbody", "tr", "td", "th"],
    ALLOWED_ATTR: ["style"],
  })
  document.body.appendChild(div)
  try {
    return await domToPng(div, { scale: pixelRatio, backgroundColor: "#ffffff" })
  } finally {
    document.body.removeChild(div)
  }
}

function buildDormTable(dates: string[]): string {
  // Union of all slot times across selected dates, sorted by order of first appearance
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
</script>
