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

    <DormGrid :week-dates="weekDates" @open-import="onOpenImport" />

    <SettingsFab style="bottom:24px;right:24px;" @click="router.push('/dorm/manage/slots')">
      <template #icon>
        <Time theme="outline" :size="22" :strokeWidth="3" />
      </template>
    </SettingsFab>

    <n-modal
      v-model:show="showImportPanel"
      preset="card"
      :title="t('dormManage.importSongs')"
      :style="{ width: '90%', maxWidth: '1000px' }"
      :mask-closable="false"
      @after-leave="importDate = ''; importDefaultSlotId = undefined; importStageId = undefined"
    >
      <SongImportPanel
        v-if="importDate"
        :date="importDate"
        :default-slot-id="importDefaultSlotId"
        :stage-id="importStageId"
        @close="showImportPanel = false"
        @song-added="onSongAdded"
      />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue"
import { useRouter, useRoute } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useSongsStore } from "../stores/songs"
import { sortSongs } from "../api/songs"
import { useMessage } from "naive-ui"
import DormGrid from "../components/dorm/DormGrid.vue"
import SongImportPanel from "../components/dorm/SongImportPanel.vue"
import SettingsFab from "../components/common/SettingsFab.vue"
import { Left, Right, Time } from "@icon-park/vue-next"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"

dayjs.extend(isoWeek)

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const settingsStore = useSettingsStore()
const songsStore = useSongsStore()
const message = useMessage()

const currentWeek = ref(dayjs().add(1, "week").startOf("isoWeek"))
const showImportPanel = ref(false)
const importDate = ref("")
const importDefaultSlotId = ref<string | undefined>(undefined)
const importStageId = ref<string | undefined>(undefined)
const refreshing = ref(false)

// um-react 桥接脚本通过 /#/dorm/manage?stage=xxx 打开本页并导入暂存音频
watch(
  () => route.query.stage,
  (v) => {
    if (!v) return
    importDate.value = currentWeek.value.startOf("isoWeek").format("YYYY-MM-DD")
    importDefaultSlotId.value = undefined
    importStageId.value = String(v)
    showImportPanel.value = true
    router.replace({ query: { ...route.query, stage: undefined } })
  },
  { immediate: true },
)

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

function onOpenImport(payload: { date: string; timeSlotId?: string }) {
  importDate.value = payload.date
  importDefaultSlotId.value = payload.timeSlotId
  showImportPanel.value = true
}

function onSongAdded() {
  songsStore.fetchSongs("dorm")
}

onMounted(() => {
  settingsStore.fetchSettings()
})
</script>
