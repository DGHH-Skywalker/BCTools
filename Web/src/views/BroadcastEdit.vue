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
    </n-space>

    <BroadcastGrid :week-dates="weekDates" :column-map="settingsStore.broadcastColumnMap" />

    <SettingsFab style="bottom:24px;right:24px;" @click="router.push('/settings/broadcast')">
      <template #icon>
        <Calendar theme="outline" :size="22" :strokeWidth="3" />
      </template>
    </SettingsFab>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import BroadcastGrid from "../components/broadcast/BroadcastGrid.vue"
import SettingsFab from "../components/ui/SettingsFab.vue"
import { Left, Right, Calendar } from "@icon-park/vue-next"
import { dayjs } from "../utils/datetime"


const { t } = useI18n()
const settingsStore = useSettingsStore()

const currentWeek = ref(dayjs().startOf("isoWeek"))
const router = useRouter()

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

onMounted(() => {
  settingsStore.fetchSettings()
})
</script>
