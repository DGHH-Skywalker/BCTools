<template>
  <div v-if="showMobileWarning" style="display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:100vh;padding:16px;box-sizing:border-box;">
    <n-image :src="logoSrc" style="width:120px;height:120px;" preview-disabled :img-props="{ draggable: false }" />
    <n-h1 style="margin-top:24px;margin-bottom:0;">{{ t("organize.title") }}</n-h1>
    <n-p style="max-width:320px;text-align:center;margin-top:12px;">{{ t("organize.mobileWarning") }}</n-p>
    <n-space style="margin-top:24px;">
      <n-button type="primary" @click="warningSkipped = true">{{ t("organize.confirmEnter") }}</n-button>
      <n-button @click="router.back()">{{ t("organize.goBack") }}</n-button>
    </n-space>
  </div>

  <div v-else style="padding:16px;max-width:1000px;margin:0 auto;">
    <PageTitle :title="t('organize.title')" />

    <n-card style="margin-bottom:16px;">
      <n-space vertical size="large">
        <n-space align="center" wrap>
          <n-text>{{ t("organize.targetDir") }}:</n-text>
          <n-input :value="targetDir || ''" :placeholder="t('organize.noTargetDir')" style="width:300px" disabled />
          <n-button @click="browse">{{ t("organize.browse") }}</n-button>
        </n-space>

        <n-space align="center" wrap>
          <n-text>{{ t("organize.year") }}:</n-text>
          <n-select v-model:value="selectedYear" :options="yearOptions" style="width:120px;" />
        </n-space>

        <n-space vertical style="width:100%;">
          <n-text>{{ t("organize.selectWeeks") }}:</n-text>
          <n-checkbox-group v-model:value="selectedWeeks">
            <n-space wrap>
              <n-checkbox v-for="opt in weekOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</n-checkbox>
            </n-space>
          </n-checkbox-group>
        </n-space>
      </n-space>
    </n-card>

    <template v-if="selectedWeeks.length > 0">
      <n-card v-for="week in selectedWeeks" :key="week" style="margin-bottom:16px;" :title="t('organize.weekDormPlaylist', { week })">
        <div v-html="tableHTMLForWeek(week)" />
        <n-space style="margin-top:12px;">
          <n-button type="primary" :disabled="!targetDir" :loading="copyingWeek === week" @click="copyWeek(week)">{{ t("organize.copyToSD") }}</n-button>
        </n-space>
      </n-card>
    </template>

    <n-empty v-else :description="t('organize.noWeeksSelected')" />

    <n-modal v-model:show="showConfirm" preset="dialog" :title="t('organize.confirmOverwrite')" positive-text="确认覆盖" negative-text="取消" @positive-click="doPendingCopy(true)">
      <n-ul>
        <n-li v-for="f in existingFiles" :key="f">{{ f }}</n-li>
      </n-ul>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { selectDir, organizeFiles } from "../api/files"
import { useMessage } from "naive-ui"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import PageTitle from "../components/common/PageTitle.vue"
import { buildDormTableHTML } from "../utils/playlistTable"
import type { TimeSlot } from "../api/types"

dayjs.extend(isoWeek)

const router = useRouter()

function isTouchDevice() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0
}
function isMobileUA() {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
}
const shouldWarn = isMobileUA() || (isTouchDevice() && !matchMedia("(pointer: fine)").matches)
const warningSkipped = ref(false)
const showMobileWarning = computed(() => shouldWarn && !warningSkipped.value)

const logoSrc = computed(() =>
  matchMedia("(prefers-color-scheme: dark)").matches ? "/logo-white.png" : "/logo.png"
)

const { t, weekdayShortName } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const message = useMessage()

const targetDir = ref("")
const selectedYear = ref(dayjs().year())
const selectedWeeks = ref<number[]>([])
const copyingWeek = ref<number | null>(null)
const showConfirm = ref(false)
const existingFiles = ref<string[]>([])
const pendingWeek = ref<number | null>(null)

const yearOptions = computed(() => {
  const current = dayjs().year()
  const years: { label: string; value: number }[] = []
  for (let y = current - 2; y <= current + 5; y++) {
    years.push({ label: `${y}`, value: y })
  }
  return years
})

interface WeekOption {
  label: string
  value: number
}

const weekOptions = computed<WeekOption[]>(() => {
  const year = selectedYear.value
  const options: WeekOption[] = []
  let cur = dayjs(`${year}-01-01`).startOf("isoWeek")
  while (cur.isoWeekYear() < year) {
    cur = cur.add(1, "week")
  }
  while (cur.isoWeekYear() === year) {
    const week = cur.isoWeek()
    const start = cur.startOf("isoWeek")
    const last = cur.endOf("isoWeek")
    options.push({
      label: `第${week}周 ${start.format("MM.DD")}-${last.format("MM.DD")}`,
      value: week,
    })
    cur = cur.add(1, "week")
  }
  return options
})

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function slotsForDate(dateStr: string): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return settingsStore.timeSlots.filter(s => s.dayIndex === dayIdx).sort((a, b) => a.order - b.order)
}

function datesForWeek(week: number): string[] {
  const start = (dayjs() as any).isoWeekYear(selectedYear.value).isoWeek(week).startOf("isoWeek")
  const dates: string[] = []
  let cur = start
  for (let i = 0; i < 7; i++) {
    dates.push(cur.format("YYYY-MM-DD"))
    cur = cur.add(1, "day")
  }
  return dates
}

function tableHTMLForWeek(week: number): string {
  const dates = datesForWeek(week)
  return buildDormTableHTML(dates, {
    title: null,
    songs: songsStore.dormSongs,
    timeSlots: settingsStore.timeSlots,
    weekdayShortName,
  })
}

function buildEntries(dates: string[]): { source: string; targetName: string }[] {
  const entries: { source: string; targetName: string }[] = []
  let seq = 1
  for (const date of dates) {
    const slots = slotsForDate(date)
    for (const slot of slots) {
      const song = songsStore.dormSongs.find(s => s.date === date && s.timeSlotId === slot.id)
      entries.push({
        source: song?.filePath || "",
        targetName: `${String(seq).padStart(2, "0")}.mp3`,
      })
      seq++
    }
  }
  return entries
}

async function browse() {
  try {
    const path = await selectDir()
    if (path) {
      targetDir.value = path
    }
  } catch (err: any) {
    if (err?.response?.status === 403 || err?.message?.includes("cancel")) return
    message.error(t("organize.selectFolderFailed"))
  }
}

async function copyWeek(week: number) {
  if (!targetDir.value) {
    message.warning(t("organize.targetDirRequired"))
    return
  }
  pendingWeek.value = week
  await doPendingCopy(false)
}

async function doPendingCopy(confirm: boolean) {
  const week = pendingWeek.value
  if (week == null) return
  const dates = datesForWeek(week)
  const entries = buildEntries(dates)
  if (entries.length === 0) {
    message.warning(t("organize.noSongsThisWeek"))
    copyingWeek.value = null
    pendingWeek.value = null
    return
  }

  copyingWeek.value = week
  try {
    const res = await organizeFiles({
      entries,
      targetDir: targetDir.value,
      mode: "copy",
      confirm,
    })
    if (res.confirmNeeded) {
      existingFiles.value = res.existingFiles || []
      showConfirm.value = true
      copyingWeek.value = null
      return
    }
    message.success(t("organize.success", { count: res.successful.length }))
    if (res.failed?.length > 0) {
      message.error(t("organize.failed", { count: res.failed.length }))
    }
  } catch (err: any) {
    message.error(err?.message || t("organize.organizeFailed"))
  } finally {
    copyingWeek.value = null
    pendingWeek.value = null
  }
}

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
})
</script>
