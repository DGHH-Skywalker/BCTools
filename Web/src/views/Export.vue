<template>
  <div class="export-page">
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
      <ExportActions
        :exporting="exporting"
        :can-export-dorm="canExport('dorm')"
        :can-export-broadcast="canExport('broadcast')"
        @export-dorm="exportPlaylist('dorm')"
        @export-broadcast="exportPlaylist('broadcast')"
      />
    </n-space>

    <SettingsFab @click="router.push('/settings/export')">
      <template #icon>
        <Setting theme="outline" :size="22" :strokeWidth="3" />
      </template>
    </SettingsFab>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, computed, h, nextTick } from "vue"
import { useI18n } from "../i18n"
import { useRouter } from "vue-router"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useExportStore } from "../stores/export"
import { useExportImage } from "../composables/useExportImage"
import WeekTransferPanel from "../components/export/WeekTransferPanel.vue"
import YearSelect from "../components/common/YearSelect.vue"
import ExportActions from "../components/export/ExportActions.vue"
import { dayjs } from "../utils/datetime"
import { useMessage } from "naive-ui"
import type { SongType } from "../api/types"
import type { TransferRenderSourceList } from "naive-ui"
import { Setting } from "@icon-park/vue-next"
import SettingsFab from "../components/ui/SettingsFab.vue"


interface DayOption {
  label: string
  value: string
}

const { t, weekdayName } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const exportStore = useExportStore()
const message = useMessage()
const { generateImage, getExportWeek } = useExportImage()
const selectedYear = ref(dayjs().year())
const exporting = ref(false)
const router = useRouter()

function canExport(type: SongType) {
  if (exportStore.selectedDates.length === 0) return false
  if (exportStore.simpleMode) return true
  return !!exportStore.backgroundImageFor(type)
}

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), songsStore.fetchSongs("broadcast"), settingsStore.fetchSettings()])
  scrollToCurrentWeek()
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
  while (cur.isBefore(end) || cur.isSame(end, "day")) {
    dates.push(cur.format("YYYY-MM-DD"))
    cur = cur.add(1, "day")
  }
  exportStore.setDates(dates)
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
  const map = new Map(transferOptions.value.map((o) => [o.value, o]))
  return exportStore.selectedDates.map((d) =>
    map.get(d) || {
      label: `${dayjs(d).format("MM.DD")} ${weekdayName(dayIndexFromDate(d))}`,
      value: d,
    }
  )
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

async function exportPlaylist(type: SongType) {
  const dates = exportStore.selectedDates
  if (dates.length === 0) return
  if (type === "dorm" && (settingsStore.timeSlots || []).length === 0) {
    message.warning(t("export.noSlots"))
    return
  }
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
    const msg = err instanceof Error && err.message ? err.message : t("export.error")
    message.error(msg)
    console.error(err)
  } finally {
    exporting.value = false
  }
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

.export-transfer {
  flex: 1 1 auto;
  min-height: 360px;
  height: calc(100vh - 200px);
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
    height: calc(100vh - 200px);
  }
}
</style>
