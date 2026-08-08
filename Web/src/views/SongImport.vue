<template>
  <div style="padding:16px;max-width:1200px;margin:0 auto;">
    <PageTitle :title="t('songImport.selectDate')">
      <template #prefix>
        <n-button text @click="showCalendar=true" v-if="!showCalendar">
          <template #icon><Left theme="outline" :size="14" :strokeWidth="3" /></template>
          {{ t("songImport.backToCalendar") }}
        </n-button>
      </template>
    </PageTitle>

    <template v-if="showCalendar">
      <CalendarGrid
        :year="year"
        :month="month"
        :songs-map="songsMap"
        selection-mode="single"
        size="mini"
        @date-click="onDateClick"
        @update:year="year = $event"
        @update:month="month = $event"
      />
    </template>

    <template v-else>
      <n-alert v-if="error" type="error" closable @close="error=''" style="margin-bottom:12px;">{{ error }}</n-alert>
      <n-alert v-if="noSlots" type="warning" style="margin-bottom:12px;">{{ t("songImport.noSlots") }}</n-alert>

      <n-grid cols="1 s:5" :x-gap="16" :y-gap="16" style="margin-bottom:16px;">
        <n-grid-item :span="3">
          <ImportExistingSongs
            :title="existingTitle"
            :songs="existingSongs"
            :slot-name="slotName"
            :slot-options="slotOptions"
            @preview="previewSong"
            @update-slot="onUpdateSlot"
          />
        </n-grid-item>
        <n-grid-item :span="2">
          <ImportUploadZone
            :title="t('songImport.uploadTitle')"
            accept=".ncm,.mp3,.mp4,.m4a,.flac,.wav,.aac,.ogg,.wma,.ape"
            formats="支持 ncm / mp3 / m4a / flac / wav / aac / ogg / wma / ape / mp4"
            @change="handleFiles"
          />
        </n-grid-item>
      </n-grid>

      <ImportProcessingGrid
        :files="processingFiles"
        :done-count="doneCount"
        :slot-options="slotOptions"
        :render-slot-label="renderSlotLabel"
        :status-tag-type="statusTagType"
        @retry="retryFile"
        @update-title="updateTitle"
        @save-metadata="saveMetadata"
        @assign-slot="onAssignSlot"
        @delete="onDeleteFile"
      />

      <n-empty v-if="processingFiles.length === 0" :description="t('songImport.empty')" style="margin-top:40px;" />
    </template>

    <n-modal
      v-model:show="showImportModal"
      preset="card"
      :title="t('songImport.importDecryptedTitle')"
      :mask-closable="false"
      :close-on-esc="false"
      style="max-width:460px;"
    >
      <n-space vertical :size="16">
        <div v-if="importFileName" style="word-break:break-all;">
          <div style="color:var(--text-color-3);font-size:13px;">{{ t("songImport.importDecryptedFile") }}</div>
          <div style="font-weight:600;margin-top:4px;">{{ importFileName }}</div>
        </div>
        <div>
          <div style="margin-bottom:6px;">{{ t("songImport.importDecryptedDate") }}</div>
          <n-date-picker v-model:value="importDate" type="date" style="width:100%;" />
        </div>
        <n-alert v-if="importError" type="error" :show-icon="false">{{ importError }}</n-alert>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button :disabled="importLoading" @click="cancelImportFromStage">{{ t("common.cancel") }}</n-button>
          <n-button
            type="primary"
            :loading="importLoading"
            :disabled="!importDate"
            @click="confirmImportFromStage"
          >{{ t("songImport.importDecryptedConfirm") }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { useFileProcessor } from "../composables/useFileProcessor"
import { Left } from "@icon-park/vue-next"
import CalendarGrid from "../components/calendar/CalendarGrid.vue"
import PageTitle from "../components/common/PageTitle.vue"
import ImportUploadZone from "../components/song-import/ImportUploadZone.vue"
import ImportExistingSongs from "../components/song-import/ImportExistingSongs.vue"
import ImportProcessingGrid from "../components/song-import/ImportProcessingGrid.vue"
import { useAudioPlayer } from "../composables/useAudioPlayer"
import { fetchStageMeta, fetchStageFile, deleteStage, markStageImported } from "../api/decrypt"
import { useRoute, useRouter } from "vue-router"
import dayjs from "dayjs"
import type { SelectOption } from "naive-ui"

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const player = useAudioPlayer()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const error = ref("")
const showCalendar = ref(true)
const year = ref(dayjs().year())
const month = ref(dayjs().month() + 1)
const selectedDate = ref("")

// Decrypted-audio import: triggered by ?stage=<id> when arriving from the
// um-react bridge. The bridge stages the decrypted file on the backend, then
// navigates here; we open a modal to pick a day and feed the file into the
// normal import pipeline.
const showImportModal = ref(false)
const importStageId = ref("")
const importFileName = ref("")
const importDate = ref<number | null>(dayjs().startOf("day").valueOf())
const importLoading = ref(false)
const importError = ref("")

const {
  processingFiles,
  doneCount,
  statusTagType,
  handleFiles,
  retryFile,
  updateTitle,
  saveMetadata,
  assignSlot,
  deleteFile,
  clearFiles,
} = useFileProcessor(selectedDate)

const noSlots = computed(() => {
  if (!selectedDate.value) return false
  const dayIdx = dayIndexFromDate(selectedDate.value)
  return !settingsStore.timeSlots.some((s) => s.dayIndex === dayIdx)
})

const selectedDayIndex = computed(() => (selectedDate.value ? dayIndexFromDate(selectedDate.value) : null))

const availableSlots = computed(() => {
  if (!selectedDayIndex.value) return []
  return settingsStore.timeSlots
    .filter((s) => s.dayIndex === selectedDayIndex.value)
    .sort((a, b) => a.order - b.order)
})

const slotOptions = computed<SelectOption[]>(() => {
  return availableSlots.value.map((s) => ({
    label: s.time,
    value: s.id,
  }))
})

const existingTitle = computed(() => `当日已点歌曲 (${existingSongs.value.length})`)
const existingSongs = computed(() => {
  if (!selectedDate.value) return []
  return songsStore.dormSongs.filter((s) => s.date === selectedDate.value)
})

const songsMap = computed(() => {
  const map: Record<string, { totalSlots: number; filledSlots: number }> = {}
  for (const song of songsStore.dormSongs) {
    if (!map[song.date]) {
      const dayIdx = dayIndexFromDate(song.date)
      const total = settingsStore.timeSlots.filter((s) => s.dayIndex === dayIdx).length
      map[song.date] = { totalSlots: total, filledSlots: 0 }
    }
    map[song.date].filledSlots++
  }
  return map
})

function slotName(slotId: string | null) {
  if (!slotId) return ""
  const slot = settingsStore.timeSlots.find((s) => s.id === slotId)
  return slot ? slot.time : "未知时段"
}

function renderSlotLabel(option: SelectOption) {
  return option.label as string
}

function previewSong(song: { filePath: string; title: string }) {
  player.play(song.filePath, song.title)
}

// 从宿舍歌单时间轴的「添加歌曲」按钮跳转过来时，URL 带 ?date=YYYY-MM-DD，
// 直接进入该日的点歌详情，无需再在日历上点选。
function applyDateQuery() {
  const d = (route.query.date as string) || ""
  if (d && /^\d{4}-\d{2}-\d{2}$/.test(d)) {
    selectedDate.value = d
    showCalendar.value = false
    clearFiles()
  }
}

onMounted(async () => {
  try {
    await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
  } catch {
    error.value = "加载数据失败"
  }
  // Arrived from the um-react import bridge with a staged decrypted file.
  const stageId = (route.query.stage as string) || ""
  if (stageId) {
    openImportFromStage(stageId)
  } else {
    applyDateQuery()
  }
})

// 同一路由复用该组件，切换 ?date= 时 onMounted 不会再触发，改用 watch。
watch(
  () => route.query.date,
  () => {
    if (!showImportModal.value) applyDateQuery()
  }
)

// The um-react bridge opens the import page in a single reused tab. When a
// second song is imported, the bridge navigates this already-mounted tab to a
// new ?stage= query; onMounted won't fire again, so watch the query instead.
watch(
  () => route.query.stage,
  (stage) => {
    const id = (stage as string) || ""
    if (id && id !== importStageId.value) {
      openImportFromStage(id)
    }
  }
)

// openImportFromStage loads the staged file's metadata and opens the import
// modal. The file bytes are only fetched on confirm to avoid a second transfer
// if the user cancels.
async function openImportFromStage(stageId: string) {
  importStageId.value = stageId
  importError.value = ""
  importFileName.value = ""
  importDate.value = dayjs().startOf("day").valueOf()
  // Reset any previous import's processing grid (the um-react bridge reuses this
  // tab for successive imports).
  clearFiles()
  try {
    const meta = await fetchStageMeta(stageId)
    importFileName.value = meta.filename
  } catch {
    importError.value = t("songImport.importDecryptedExpired")
  }
  showImportModal.value = true
}

async function confirmImportFromStage() {
  if (!importStageId.value || !importDate.value) return
  importLoading.value = true
  importError.value = ""
  try {
    const blob = await fetchStageFile(importStageId.value)
    const file = new File([blob], importFileName.value || "decrypted.mp3")
    // Jump to the chosen day's import detail, then feed the staged file into the
    // existing import pipeline (convert/stash + create song for that date).
    selectedDate.value = dayjs(importDate.value).format("YYYY-MM-DD")
    showCalendar.value = false
    // Wait for conversion + song creation so we know whether it succeeded before
    // telling um-react to drop the card.
    const items = await handleFiles({ fileList: [file] })
    const succeeded = items.some((it) => it.status === "done")
    if (succeeded) {
      // Signal success back to um-react through the backend. um-react polls this
      // flag, removes the decrypted card via its own delete logic, then deletes
      // the stage. Do NOT delete the stage here — um-react cleans it up.
      markStageImported(importStageId.value).catch(() => {})
    }
    showImportModal.value = false
    importStageId.value = ""
    // Clear the query so a refresh does not re-trigger. The um-react bridge
    // reuses this tab for the next import; a watcher picks up the new stage.
    await router.replace({ query: {} })
  } catch (err: any) {
    importError.value = err?.message || "导入失败"
  } finally {
    importLoading.value = false
  }
}

function cancelImportFromStage() {
  showImportModal.value = false
  if (importStageId.value) {
    deleteStage(importStageId.value).catch(() => {})
    importStageId.value = ""
  }
  router.replace({ query: {} }).catch(() => {})
}

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function onDateClick(date: string) {
  if (!date) return
  selectedDate.value = date
  showCalendar.value = false
  clearFiles()
}

async function onAssignSlot(file: any, slotId: string) {
  const err = await assignSlot(file, slotId)
  if (err) {
    error.value = err
  }
}

async function onDeleteFile(file: any) {
  const err = await deleteFile(file)
  if (err) {
    error.value = err
  }
}

async function onUpdateSlot(songId: number, slotId: string) {
  try {
    await songsStore.assignSong(songId, slotId)
  } catch (err: any) {
    error.value = err?.message || "调整时段失败"
  }
}
</script>
