<template>
  <div class="export-page">
    <PageTitle :title="t('export.title')">
      <template #extra>
        <n-button quaternary circle @click="showAdvancedDrawer = true">
          <template #icon>
            <Setting theme="outline" :size="20" :strokeWidth="3" />
          </template>
        </n-button>
      </template>
    </PageTitle>

    <n-space align="center" justify="space-between" wrap style="margin-bottom:12px;">
      <n-space align="center" wrap>
        <n-text strong>{{ t("export.sourceTitle") }}</n-text>
        <YearSelect v-model:value="selectedYear" />
      </n-space>
      <n-space align="center" wrap>
        <n-text strong>{{ t("export.targetTitle") }}</n-text>
        <n-button-group>
          <n-button size="small" dashed @click="quickSelect(0)">{{ t("export.thisWeek") }}</n-button>
          <n-button size="small" dashed @click="quickSelect(-1)">{{ t("export.lastWeek") }}</n-button>
          <n-button size="small" dashed @click="quickSelect(1)">{{ t("export.nextWeek") }}</n-button>
        </n-button-group>
      </n-space>
    </n-space>

    <n-transfer
      v-model:value="exportStore.selectedDates"
      :options="transferOptions"
      :render-source-list="renderSourceList"
      :render-target-list="renderTargetList"
      source-filterable
      target-filterable
      :source-title="t('export.sourceTitle')"
      :target-title="t('export.targetTitle')"
      class="export-transfer"
    />

    <n-space wrap style="margin-top:8px;">
      <n-tag v-for="d in exportStore.selectedDates" :key="d" closable @close="exportStore.removeDate(d)">{{ d }}</n-tag>
    </n-space>

    <n-space style="margin-top:16px;" v-if="exportStore.selectedDates.length > 0" wrap>
      <n-tooltip v-if="!canExport('dorm')" trigger="hover">
        <template #trigger>
          <n-button type="primary" dashed disabled>{{ t("export.exportDorm") }}</n-button>
        </template>
        {{ t("export.needBackground") }}
      </n-tooltip>
      <n-button v-else type="primary" dashed @click="exportPlaylist('dorm')" :loading="exporting">
        {{ t("export.exportDorm") }}
      </n-button>

      <n-tooltip v-if="!canExport('broadcast')" trigger="hover">
        <template #trigger>
          <n-button type="primary" dashed disabled>{{ t("export.exportBroadcast") }}</n-button>
        </template>
        {{ t("export.needBackground") }}
      </n-tooltip>
      <n-button v-else type="primary" dashed @click="exportPlaylist('broadcast')" :loading="exporting">
        {{ t("export.exportBroadcast") }}
      </n-button>
    </n-space>

    <n-drawer
      v-model:show="showAdvancedDrawer"
      placement="right"
      :width="drawerWidth"
      :native-scrollbar="false"
      resizable
    >
      <n-scrollbar style="height:100%;">
        <n-space align="center" justify="space-between" style="padding:16px 16px 0;">
          <n-text strong style="font-size:16px;">{{ t('export.advancedSettings') }}</n-text>
          <n-button text circle @click="showAdvancedDrawer = false">
            <template #icon>
              <Close theme="outline" :size="20" :strokeWidth="3" />
            </template>
          </n-button>
        </n-space>
        <n-space vertical size="large" style="padding:16px;">
          <n-space align="center" wrap style="width:100%;">
            <n-text strong>{{ t("export.mode") }}</n-text>
            <n-switch v-model:value="exportStore.simpleMode" :rail-style="switchRailStyle">
              <template #checked>{{ t("export.simpleMode") }}</template>
              <template #unchecked>{{ t("export.posterMode") }}</template>
            </n-switch>
          </n-space>

          <n-space v-if="!exportStore.simpleMode" align="center" wrap style="width:100%;">
            <n-upload :default-upload="false" :show-file-list="false" accept="image/*" @change="(o) => onBackgroundUploadChange(o, 'dorm')">
              <div
                class="bg-preview-card"
                :style="{ backgroundImage: exportStore.backgroundImageDorm ? `url(${exportStore.backgroundImageDorm})` : 'none' }"
              >
                <div v-if="!exportStore.backgroundImageDorm" class="bg-preview-placeholder">{{ t("export.dorm") }}{{ t("export.backgroundImage") }}</div>
                <div v-else class="bg-preview-overlay">{{ t("common.replace") }}</div>
              </div>
            </n-upload>

            <n-upload :default-upload="false" :show-file-list="false" accept="image/*" @change="(o) => onBackgroundUploadChange(o, 'broadcast')">
              <div
                class="bg-preview-card"
                :style="{ backgroundImage: exportStore.backgroundImageBroadcast ? `url(${exportStore.backgroundImageBroadcast})` : 'none' }"
              >
                <div v-if="!exportStore.backgroundImageBroadcast" class="bg-preview-placeholder">{{ t("export.broadcast") }}{{ t("export.backgroundImage") }}</div>
                <div v-else class="bg-preview-overlay">{{ t("common.replace") }}</div>
              </div>
            </n-upload>
          </n-space>

          <template v-if="!exportStore.simpleMode">
            <n-space vertical>
              <n-text strong>{{ t("export.vacationBadge") }}</n-text>
              <n-radio-group v-model:value="exportStore.vacationBadge" vertical>
                <n-radio value="none">{{ t("export.badgeNone") }}</n-radio>
                <n-radio value="summer">{{ t("export.badgeSummer") }}</n-radio>
                <n-radio value="winter">{{ t("export.badgeWinter") }}</n-radio>
              </n-radio-group>
            </n-space>

            <n-space align="start" style="width:100%;">
              <n-space vertical style="flex:1;">
                <n-text strong>{{ t("export.posterQuoteDorm") }}</n-text>
                <n-input
                  v-model:value="exportStore.posterQuoteDorm"
                  type="textarea"
                  :placeholder="t('export.posterQuoteDormPlaceholder')"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  style="width:100%;"
                />
              </n-space>
              <n-space vertical style="flex:1;">
                <n-text strong>{{ t("export.posterQuoteBroadcast") }}</n-text>
                <n-input
                  v-model:value="exportStore.posterQuoteBroadcast"
                  type="textarea"
                  :placeholder="t('export.posterQuoteBroadcastPlaceholder')"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  style="width:100%;"
                />
              </n-space>
            </n-space>

            <n-space align="start" style="width:100%;">
              <n-space vertical style="flex:1;">
                <n-text strong>{{ t("export.posterBlurDorm") }}: {{ blurLabel('dorm') }}</n-text>
                <div class="blur-preview" :style="blurPreviewStyle('dorm')">
                  <span v-if="!exportStore.backgroundImageDorm" class="blur-placeholder">{{ t("export.blurPlaceholder") }}</span>
                </div>
                <n-slider v-model:value="exportStore.posterBlurDorm" :min="0" :max="20" :step="0.5" />
              </n-space>
              <n-space vertical style="flex:1;">
                <n-text strong>{{ t("export.posterBlurBroadcast") }}: {{ blurLabel('broadcast') }}</n-text>
                <div class="blur-preview" :style="blurPreviewStyle('broadcast')">
                  <span v-if="!exportStore.backgroundImageBroadcast" class="blur-placeholder">{{ t("export.blurPlaceholder") }}</span>
                </div>
                <n-slider v-model:value="exportStore.posterBlurBroadcast" :min="0" :max="20" :step="0.5" />
              </n-space>
            </n-space>
          </template>
        </n-space>
      </n-scrollbar>
    </n-drawer>

  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed, h, nextTick } from "vue"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import { useAppConfig } from "../composables/useAppConfig"
import WeekTransferPanel from "../components/export/WeekTransferPanel.vue"
import PageTitle from "../components/common/PageTitle.vue"
import YearSelect from "../components/common/YearSelect.vue"
import { buildDormTableHTML } from "../utils/playlistTable"

interface DayOption {
  label: string
  value: string
}
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import { useMessage } from "naive-ui"
import { domToPng } from "modern-screenshot"
import DOMPurify from "dompurify"
import type { Song, SongType, TimeSlot } from "../api/types"
import type { UploadFileInfo, TransferRenderSourceList } from "naive-ui"
import { Setting, Close } from "@icon-park/vue-next"

dayjs.extend(isoWeek)

const { t, weekdayName, weekdayShortName } = useI18n()
const { config } = useAppConfig()
const themeColor = computed(() => config.value.themeColor)

const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const exportStore = useExportStore()
const message = useMessage()
const selectedYear = ref(dayjs().year())
const exporting = ref(false)
const showAdvancedDrawer = ref(false)

const fontBase64Cache: Record<string, string | null> = {}
const imageBase64Cache: Record<string, string | null> = {}

function canExport(type: SongType) {
  if (exportStore.selectedDates.length === 0) return false
  if (exportStore.simpleMode) return true
  return !!exportStore.backgroundImageFor(type)
}
const drawerWidth = ref(window.innerWidth <= 768 ? "100%" : "50%")
function updateDrawerWidth() {
  drawerWidth.value = window.innerWidth <= 768 ? "100%" : "50%"
}
function blurLabel(type: SongType) {
  const blur = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
  return `${blur.toFixed(1)}px`
}

function blurPreviewStyle(type: SongType) {
  const bg = exportStore.backgroundImageFor(type)
  const blur = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
  const style: Record<string, string> = {
    backgroundImage: bg ? `url(${bg})` : "none",
    backgroundSize: "cover",
    backgroundPosition: "center",
    filter: `blur(${blur}px) brightness(0.95)`,
  }
  if (!bg) {
    style.background = themeColor.value
  }
  return style
}

function switchRailStyle({ checked }: { checked: boolean }) {
  return {
    background: checked ? "#1a1a1a" : themeColor.value,
  }
}

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), songsStore.fetchSongs("broadcast"), settingsStore.fetchSettings()])
  updateDrawerWidth()
  window.addEventListener("resize", updateDrawerWidth)
  scrollToCurrentWeek()
})

onUnmounted(() => {
  window.removeEventListener("resize", updateDrawerWidth)
})

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function scrollToCurrentWeek() {
  nextTick(() => {
    const el = document.getElementById("week-current")
    if (el) {
      el.scrollIntoView({ behavior: "auto", block: "center" })
    }
  })
}

watch(selectedYear, () => {
  scrollToCurrentWeek()
})

function quickSelect(offset: number) {
  const d = dayjs().add(offset, "week")
  const start = d.startOf("isoWeek")
  const end = d.endOf("isoWeek")
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

function onBackgroundUploadChange({ fileList }: { fileList: UploadFileInfo[] }, type: "dorm" | "broadcast") {
  const file = fileList[0]?.file as File
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    exportStore.setBackgroundImage(type, reader.result as string)
  }
  reader.onerror = () => {
    message.error(t("export.backgroundImageError"))
  }
  reader.readAsDataURL(file)
}

function transferOptionsForYear(year: number): DayOption[] {
  const options: DayOption[] = []
  let cur = dayjs(`${year}-01-01`).startOf("isoWeek")
  const end = dayjs(`${year}-12-31`).endOf("isoWeek")
  while (cur.isBefore(end) || cur.isSame(end, "day")) {
    const date = cur.format("YYYY-MM-DD")
    options.push({
      label: `${cur.format("MM.DD")} ${weekdayName(dayIndexFromDate(date))}`,
      value: date,
    })
    cur = cur.add(1, "day")
  }
  return options
}

const transferOptions = computed(() => transferOptionsForYear(selectedYear.value))
const targetOptions = computed<DayOption[]>(() => {
  const map = new Map(transferOptions.value.map(o => [o.value, o]))
  return exportStore.selectedDates
    .map(d => map.get(d) || {
      label: `${dayjs(d).format("MM.DD")} ${weekdayName(dayIndexFromDate(d))}`,
      value: d,
    })
})

const renderSourceList: TransferRenderSourceList = ({ onCheck, checkedOptions, pattern }) => {
  return h(WeekTransferPanel, {
    options: transferOptions.value,
    selectedOptions: checkedOptions as DayOption[],
    pattern,
    onUpdate: onCheck as (vals: string[]) => void,
  })
}

const renderTargetList: TransferRenderSourceList = ({ onCheck, checkedOptions, pattern }) => {
  return h(WeekTransferPanel, {
    options: targetOptions.value,
    selectedOptions: checkedOptions as DayOption[],
    pattern,
    onUpdate: onCheck as (vals: string[]) => void,
  })
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

function getExportWeek(dates: string[]): number {
  if (dates.length === 0) return dayjs().isoWeek()
  const earliest = [...dates].sort()[0]
  return dayjs(earliest).isoWeek()
}

async function exportPlaylist(type: SongType) {
  const dates = exportStore.selectedDates
  if (dates.length === 0) return
  exporting.value = true
  try {
    const dataUrl = await generateImage(dates, type)
    const week = getExportWeek(dates)
    const a = document.createElement("a")
    a.href = dataUrl
    const title = type === "dorm" ? t("export.dorm") : t("export.broadcast")
    a.download = `第${week}周${title}.png`
    a.click()
    message.success(t("export.success"))
  } catch (err) {
    message.error(t("export.error"))
    console.error(err)
  } finally {
    exporting.value = false
  }
}

async function generateImage(dates: string[], type: SongType): Promise<string> {
  const isPoster = !exportStore.simpleMode
  let rawHtml = ""
  let bgColor = "#ffffff"
  if (isPoster) {
    rawHtml = await buildPosterHTML(dates, type)
    bgColor = "#1a1a1a"
  } else {
    rawHtml = type === "dorm"
      ? buildDormTableHTML(dates, {
          title: t("export.dorm"),
          songs: songsStore.dormSongs,
          timeSlots: settingsStore.timeSlots,
          weekdayShortName,
        })
      : buildBroadcastTable(dates)
  }

  // 动态创建临时渲染容器：外层 0×0 overflow:hidden 隐藏 UI，内层正常尺寸供 domToPng 捕获，
  // 避免 left:-9999px 导致克隆节点在 SVG viewport 外生成空白图片。
  const wrapper = document.createElement("div")
  wrapper.style.cssText = "position:fixed;left:0;top:0;width:0;height:0;overflow:hidden;pointer-events:none;z-index:-1;"
  const container = document.createElement("div")
  container.style.cssText = isPoster
    ? `position:absolute;left:0;top:0;width:1080px;min-height:1920px;height:auto;background:${bgColor};`
    : "position:absolute;left:0;top:0;width:1080px;padding:48px;background:#fff;font-family:sans-serif;height:auto;"
  container.innerHTML = DOMPurify.sanitize(rawHtml, {
    ALLOWED_TAGS: ["div", "span", "svg", "path", "rect", "circle", "polygon", "style", "h2", "table", "thead", "tbody", "tr", "td", "th", "img", "br"],
    ALLOWED_ATTR: ["style", "class", "viewBox", "fill", "stroke", "stroke-width", "cx", "cy", "r", "x", "y", "width", "height", "points", "rx", "d", "xmlns", "src", "alt"],
  })

  const fontFiles = ["江西拙楷3.0.ttf", "方正颜宋简体.ttf", "方正小标宋简.TTF", "仿宋_GB2312.ttf"]
  const fontFamilies = ["JiangxiZhuokai", "FZYanSong", "FZXiaoBiaoSong", "FangSong_GB2312"]
  const style = document.createElement("style")
  let styleContent = ""
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
        await Promise.race([
          document.fonts.load(`12px ${family}`),
          new Promise((resolve) => setTimeout(resolve, 3000)),
        ])
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
  } finally {
    document.body.removeChild(wrapper)
  }
}

function isMultiWeek(dates: string[]): boolean {
  return new Set(dates.map(d => dayjs(d).isoWeek())).size > 1
}

function dateCellHtml(date: string): string {
  const shortWd = escapeHtml(`周${weekdayShortName(dayIndexFromDate(date))}`)
  const formattedDate = escapeHtml(dayjs(date).format("MM.DD"))
  return `<div style="display:flex;flex-direction:column;align-items:center;justify-content:center;line-height:1.2;">
    <div style="font-family:'HarmonyOS Sans SC','HarmonyOS Sans','DengXian','Microsoft YaHei','PingFang SC',sans-serif;font-size:18px;">${formattedDate}</div>
    <div style="font-family:'FZXiaoBiaoSong','方正小标宋简','STSong','SimSun','Songti SC',serif;font-size:24px;">${shortWd}</div>
  </div>`
}

function buildBroadcastTable(dates: string[]): string {
  const allSongs: Song[] = []
  for (const date of dates) {
    allSongs.push(...songsStore.broadcastSongs.filter(s => s.date === date))
  }
  allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)
  const multiWeek = isMultiWeek(dates)

  const rows = allSongs.map(s =>
    `<tr><td style="padding:8px;border:1px solid #ddd;text-align:center;white-space:nowrap;font-weight:400;">${dateCellHtml(s.date)}</td>` +
    `<td style="padding:8px;border:1px solid #ddd;font-weight:400;word-break:break-word;">${escapeHtml(s.title)}</td>` +
    `<td style="padding:8px;border:1px solid #ddd;font-weight:400;word-break:break-word;">${escapeHtml(s.remark)}</td></tr>`
  ).join("")

  return `<h2 style="text-align:center;margin-bottom:24px;font-weight:400;">${escapeHtml(t("export.broadcast"))}</h2>
    <table style="width:100%;border-collapse:collapse;table-layout:fixed;">
      <colgroup><col style="width:18%;" /><col style="width:47%;" /><col style="width:35%;" /></colgroup>
      <thead><tr><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;font-weight:400;"></th><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;font-weight:400;">歌名</th><th style="padding:8px;border:1px solid #ddd;background:#f0f0f0;font-weight:400;">备注</th></tr></thead>
      <tbody>${rows}</tbody>
    </table>`
}

async function buildPosterHTML(dates: string[], type: SongType): Promise<string> {
  const titleText = escapeHtml(type === "dorm" ? t("export.dorm") : t("export.broadcast"))
  const badge = exportStore.vacationBadge
  const blurPx = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
  const quoteValue = type === "dorm" ? exportStore.posterQuoteDorm : exportStore.posterQuoteBroadcast
  const quoteText = escapeHtml(quoteValue || "人生南北多歧路 君向潇湘我向秦")

  const badgeHtml = badge === "none" ? ""
    : `<div style="position:absolute;top:90px;right:80px;display:flex;flex-direction:row;gap:10px;z-index:3;">
        <div style="font-family:'JiangxiZhuokai','Microsoft YaHei','PingFang SC',sans-serif;font-size:34px;font-weight:400;color:${themeColor.value};writing-mode:vertical-rl;text-orientation:upright;letter-spacing:0.12em;">${badge === "summer" ? "暑假" : "寒假"}</div>
        <div style="font-family:'JiangxiZhuokai','Microsoft YaHei','PingFang SC',sans-serif;font-size:34px;font-weight:400;color:${themeColor.value};writing-mode:vertical-rl;text-orientation:upright;letter-spacing:0.12em;">限定</div>
      </div>`

  const bgImage = exportStore.backgroundImageFor(type)
  const bgLayer = bgImage
    ? `<div style="position:absolute;inset:0;background-image:url(${bgImage});background-size:cover;background-position:center;filter:blur(${blurPx}px) brightness(0.95);transform:scale(1.08);"></div>`
    : `<div style="position:absolute;inset:0;background:linear-gradient(135deg,${themeColor.value} 0%,#004d70 50%,#1a1a1a 100%);"></div>`

  const tableHtml = type === "dorm"
    ? buildDormPosterTable(dates)
    : buildBroadcastPosterTableV2(dates)

  const titlePadding = "0 80px"

  const logoSrc = await loadImageBase64("/logo-white.png")
  const logoHtml = logoSrc
    ? `<img src="${logoSrc}" alt="logo" style="height:120px;width:auto;object-fit:contain;flex-shrink:0;pointer-events:none;-webkit-user-drag:none;filter:drop-shadow(0 0.08em 0.15em rgba(0,0,0,0.35));" />`
    : `<div style="width:54px;height:105px;flex-shrink:0;"></div>`

  const titleHtml = `<div style="position:absolute;left:0;right:0;top:0;height:240px;display:flex;align-items:center;justify-content:center;gap:28px;padding:${titlePadding};box-sizing:border-box;z-index:10;">
    ${logoHtml}
    <div style="font-family:'JiangxiZhuokai','STSong','SimSun','Songti SC',serif;font-size:90px;font-weight:400;color:#fff;text-shadow:0 0.08em 0.15em rgba(0,0,0,0.35);white-space:nowrap;">${titleText}</div>
  </div>`

  return `<div style="position:absolute;inset:0;overflow:hidden;z-index:0;">
    ${bgLayer}
    <div style="position:absolute;inset:0;background:rgba(0,0,0,0.08);"></div>
  </div>
  ${badgeHtml}
  <div style="position:relative;z-index:1;padding-top:260px;padding-bottom:60px;display:flex;flex-direction:column;align-items:center;gap:24px;">
    <div style="width:960px;background:rgba(0,0,0,0.32);border-radius:40px;box-shadow:0 12px 38px 0 rgba(0,0,0,0.45);overflow:hidden;display:flex;flex-direction:column;align-items:center;">
      <div style="width:100%;display:flex;flex-direction:column;align-items:center;padding-top:36px;box-sizing:border-box;">
        <div style="font-family:'FangSong_GB2312','仿宋_GB2312','FangSong','STFangsong','serif';font-size:36px;color:#d7d7d7;text-align:center;letter-spacing:0.05em;line-height:1.4;">${quoteText}</div>
        <div style="width:90%;height:1px;background:${themeColor.value};margin:24px 0 0;opacity:0.6;"></div>
      </div>
      ${tableHtml}
      <div style="width:90%;height:1px;background:${themeColor.value};margin:0 0 36px;opacity:0.6;"></div>
    </div>
  </div>
  ${titleHtml}`
}

function buildDormPosterTable(dates: string[]): string {
  const timeOrder: Record<string, number> = {}
  for (const date of dates) {
    for (const slot of slotsForDate(date)) {
      if (timeOrder[slot.time] === undefined) timeOrder[slot.time] = slot.order
    }
  }
  const times = Object.keys(timeOrder).sort((a, b) => timeOrder[a] - timeOrder[b])
  const multiWeek = isMultiWeek(dates)
  const type: SongType = "dorm"

  const colCount = times.length + 1
  const headerCells = [
    cellHtml("", true, false, 0, colCount, 0, "slot", type),
    ...times.map((time, i) => cellHtml(time, true, false, i + 1, colCount, 0, "slot", type)),
  ].join("")
  const headerRow = rowHtml(headerCells, true, 0)

  const dataRows = dates.map((date, rowIdx) => {
    const daySlots = slotsForDate(date)
    const slotMap: Record<string, Song | undefined> = {}
    for (const slot of daySlots) {
      slotMap[slot.time] = songsStore.dormSongs.find(s => s.date === date && s.timeSlotId === slot.id)
    }
    const cells = [
      cellHtml(dateCellHtml(date), false, true, 0, colCount, rowIdx + 1, "slot", type),
      ...times.map((time, i) => cellHtml(slotMap[time] ? escapeHtml(slotMap[time]!.title) : "", false, false, i + 1, colCount, rowIdx + 1, "title", type)),
    ].join("")
    return rowHtml(cells, false, rowIdx + 1)
  }).join("")

  return gridWrapperHtml(headerRow + dataRows, colCount, dates.length, true, type)
}

function buildBroadcastPosterTable(dates: string[]): string {
  const allSongs: Song[] = []
  for (const date of dates) {
    allSongs.push(...songsStore.broadcastSongs.filter(s => s.date === date))
  }
  allSongs.sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)
  const multiWeek = isMultiWeek(dates)
  const type: SongType = "broadcast"

  const headers = ["", "歌名", "备注"]
  const colCount = headers.length
  const headerCells = headers.map((h, i) => {
    const fontRole = i === 0 ? "slot" : undefined
    return cellHtml(h, true, i === 0, i, colCount, 0, fontRole, type)
  }).join("")
  const headerRow = rowHtml(headerCells, true, 0)

  const dataRows = allSongs.map((s, rowIdx) => {
    const cells = [
      cellHtml(dateCellHtml(s.date), false, true, 0, colCount, rowIdx + 1, "slot", type),
      cellHtml(escapeHtml(s.title), false, false, 1, colCount, rowIdx + 1, "title", type),
      cellHtml(escapeHtml(s.remark), false, false, 2, colCount, rowIdx + 1, undefined, type)
    ].join("")
    return rowHtml(cells, false, rowIdx + 1)
  }).join("")

  return gridWrapperHtml(headerRow + dataRows, colCount, allSongs.length, false, type)
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

  const dataRows = rowLabels.map((label, rowIdx) => {
    const cells = [
      cellHtml(label, false, true, 0, colCount, rowIdx + 1, "slot", type),
      ...dates.map((date, colIdx) => {
        const songs = songsStore.broadcastSongs.filter(s => s.date === date)
        const targetSongs = songs.filter(s => {
          const period = s.period || (dayjs(s.createdAt).hour() < 12 ? "noon" : "afternoon")
          return rowIdx === 0 ? period === "noon" : period === "afternoon"
        })
        const title = targetSongs.length > 0 ? targetSongs[0].title : ""
        return cellHtml(escapeHtml(title), false, false, colIdx + 1, colCount, rowIdx + 1, "title", type)
      }),
    ].join("")
    return rowHtml(cells, false, rowIdx + 1)
  }).join("")

  return gridWrapperHtml(headerRow + dataRows, colCount, 2, true, type)
}

function gridWrapperHtml(rowsHtml: string, colCount: number, rowCount: number, hasDateCol: boolean = false, type: SongType = "dorm"): string {
  const cols = hasDateCol ? `140px repeat(${colCount - 1}, 1fr)` : `repeat(${colCount}, 1fr)`
  const borderColor = type === "broadcast" ? "#ffffff" : "rgba(255,255,255,0.85)"
  return `<div style="width:94%;display:grid;grid-template-columns:${cols};grid-template-rows:auto repeat(${rowCount},minmax(100px,auto));gap:0;align-items:stretch;justify-items:stretch;font-family:'Microsoft YaHei','PingFang SC','Hiragino Sans GB',sans-serif;border:1px solid ${borderColor};border-radius:12px;overflow:hidden;">
    ${rowsHtml}
  </div>`
}

function rowHtml(cellsHtml: string, isHeader: boolean, rowIndex: number): string {
  return cellsHtml
}

function cellHtml(content: string, isHeader: boolean, narrow: boolean, colIndex: number, totalCols: number, rowIndex: number, fontRole?: "title" | "slot", type: SongType = "dorm"): string {
  let fontFamily = "'JiangxiZhuokai','Microsoft YaHei','PingFang SC','Hiragino Sans GB',sans-serif"
  if (fontRole === "title") {
    fontFamily = "'FZYanSong','方正颜宋简体','STSong','SimSun','Songti SC',serif"
  } else if (fontRole === "slot" || isHeader) {
    fontFamily = "'FZXiaoBiaoSong','方正小标宋简','STSong','SimSun','Songti SC',serif"
  }
  let fontSize = isHeader ? "40px" : "36px"
  let fontWeight = "400"
  let opacity = "1"
  // 日期列：小号、细等线/HarmonyOS Sans，降低存在感
  if (narrow && !isHeader) {
    fontFamily = "'HarmonyOS Sans SC','HarmonyOS Sans','DengXian','Microsoft YaHei','PingFang SC',sans-serif"
    fontSize = "24px"
    fontWeight = "100"
    opacity = "0.85"
  }
  const color = "#ffffff"
  const padding = narrow ? "0 8px" : "0 12px"
  const borderColor = type === "broadcast" ? "#ffffff" : "rgba(255,255,255,0.85)"
  // 播音歌单删除行与行之间的横向白线；宿舍歌单删除列与列之间的纵向白线
  const borderBottom = type === "broadcast" ? "" : `border-bottom:1px solid ${borderColor};`
  const borderRight = type === "dorm" || colIndex === totalCols - 1 ? "" : `border-right:1px solid ${borderColor};`
  const titleOverflow = fontRole === "title" && !isHeader ? "overflow:hidden;" : ""
  const displayContent = content
  return `<div style="display:flex;justify-content:center;align-items:center;text-align:center;font-family:${fontFamily};font-size:${fontSize};font-weight:${fontWeight};color:${color};opacity:${opacity};background:transparent;padding:${padding};line-height:1.25;box-sizing:border-box;min-height:100%;word-break:break-word;overflow-wrap:anywhere;${titleOverflow}${borderBottom}${borderRight}">${displayContent}</div>`
}
</script>

<style scoped>
.export-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  box-sizing: border-box;
  position: relative;
}

.export-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.export-transfer {
  flex: 1 1 auto;
  min-height: 360px;
  height: calc(100vh - 260px);
  margin-bottom: 12px;
}

.export-transfer :deep(.n-transfer) {
  height: 100%;
}

.export-transfer :deep(.n-transfer-list) {
  height: 100%;
}

.export-transfer :deep(.n-transfer-list-body) {
  height: calc(100% - 48px);
}

@media (max-width: 768px) {
  .export-transfer {
    min-height: 240px;
    height: calc(100vh - 240px);
  }
}

.export-page :deep(.n-upload-file-list) {
  display: none;
}

.blur-preview {
  width: 100%;
  aspect-ratio: 9 / 16;
  max-width: 180px;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
  overflow: hidden;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
}

.blur-placeholder {
  color: #fff;
  font-size: 14px;
  opacity: 0.9;
}

.bg-preview-card {
  width: 80px;
  height: 56px;
  border-radius: 6px;
  border: 1px dashed var(--theme-color, #0086c3);
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: pointer;
  position: relative;
}

.bg-preview-placeholder {
  font-size: 12px;
  color: #888;
  text-align: center;
  padding: 4px;
  line-height: 1.2;
}

.bg-preview-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.15s;
}

.bg-preview-card:hover .bg-preview-overlay {
  opacity: 1;
}
</style>
