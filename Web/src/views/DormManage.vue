<template>
  <div style="padding:16px;max-width:1600px;margin:0 auto;">
    <n-space align="center" style="margin-bottom:16px;">
      <n-button size="small" @click="prevWeek">
        <template #icon><Left theme="outline" :size="14" :strokeWidth="3" /></template>
        {{ t("export.lastWeek") }}
      </n-button>
      <n-text strong>{{ weekLabel }}</n-text>
      <n-button size="small" @click="nextWeek">
        {{ t("export.nextWeek") }}
        <template #icon><Right theme="outline" :size="14" :strokeWidth="3" /></template>
      </n-button>
      <n-button size="small" @click="goCurrentWeek">{{ t("export.thisWeek") }}</n-button>
      <n-button size="small" :loading="refreshing" @click="refreshDormSongs">刷新</n-button>
    </n-space>

    <DormGrid :week-dates="weekDates" />

    <SettingsFab style="bottom:24px;right:24px;" @click="showSlotModal = true">
      <template #icon>
        <Time theme="outline" :size="22" :strokeWidth="3" />
      </template>
    </SettingsFab>
    <TimeSlotModal v-model:show="showSlotModal" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useSongsStore } from "../stores/songs"
import { sortSongs } from "../api/songs"
import { useMessage } from "naive-ui"
import DormGrid from "../components/dorm/DormGrid.vue"
import TimeSlotModal from "../components/dorm/TimeSlotModal.vue"
import SettingsFab from "../components/common/SettingsFab.vue"
import { Left, Right, Time } from "@icon-park/vue-next"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"

dayjs.extend(isoWeek)

const { t } = useI18n()
const settingsStore = useSettingsStore()
const songsStore = useSongsStore()
const message = useMessage()

const currentWeek = ref(dayjs().startOf("isoWeek"))
const showSlotModal = ref(false)
const refreshing = ref(false)

const weekDates = computed(() => {
  const dates: string[] = []
  let cur = currentWeek.value.startOf("isoWeek")
  const end = currentWeek.value.endOf("isoWeek")
  while (cur.isBefore(end) || cur.isSame(end, "day")) {
    dates.push(cur.format("YYYY-MM-DD"))
    cur = cur.add(1, "day")
  }
  return dates
})

const weekLabel = computed(() => {
  const year = currentWeek.value.isoWeekYear()
  const week = currentWeek.value.isoWeek()
  return `${year} 年第 ${week} 周`
})

function prevWeek() { currentWeek.value = currentWeek.value.add(-1, "week") }
function nextWeek() { currentWeek.value = currentWeek.value.add(1, "week") }
function goCurrentWeek() { currentWeek.value = dayjs().startOf("isoWeek") }

async function refreshDormSongs() {
  refreshing.value = true
  try {
    await sortSongs("dorm")
    await songsStore.fetchSongs("dorm")
    message.success("已刷新排序")
  } catch {
    message.error("刷新排序失败")
  } finally {
    refreshing.value = false
  }
}

onMounted(() => {
  settingsStore.fetchSettings()
})
</script>
