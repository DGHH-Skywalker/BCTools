<template>
  <div style="padding:8px;">
    <n-alert v-if="error" type="error" closable @close="error = ''" style="margin-bottom: 12px;">{{ error }}</n-alert>
    <n-alert v-if="noSlots" type="warning" style="margin-bottom: 12px;">{{ t("songImport.noSlots") }}</n-alert>

    <n-grid cols="1 s:5" :x-gap="16" :y-gap="16" style="margin-bottom: 16px;">
      <n-grid-item :span="3">
        <ImportExistingSongs
          :title="existingTitle"
          :songs="existingSongs"
          :slot-name="slotName"
          :slot-options="slotOptions"
          @preview="previewSong"
          @update-slot="onUpdateSlot"
          @delete-song="onDeleteExistingSong"
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

    <n-empty v-if="processingFiles.length === 0" :description="t('songImport.empty')" style="margin-top: 40px;" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import { useFileProcessor } from "../../composables/useFileProcessor"
import ImportUploadZone from "../song-import/ImportUploadZone.vue"
import ImportExistingSongs from "../song-import/ImportExistingSongs.vue"
import ImportProcessingGrid from "../song-import/ImportProcessingGrid.vue"
import { useAudioPlayer } from "../../composables/useAudioPlayer"
import dayjs from "dayjs"
import type { SelectOption } from "naive-ui"

const props = defineProps<{
  date: string
  defaultSlotId?: string
}>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "song-added"): void
}>()

const { t } = useI18n()
const player = useAudioPlayer()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const error = ref("")
const selectedDate = ref(props.date)
const defaultTimeSlotId = ref<string | null>(props.defaultSlotId || null)

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
} = useFileProcessor(selectedDate, defaultTimeSlotId)

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

async function onDeleteExistingSong(songId: number) {
  try {
    await songsStore.deleteSong(songId)
  } catch (err: any) {
    error.value = err?.message || "删除失败"
  }
}

onMounted(async () => {
  try {
    await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
  } catch {
    error.value = "加载数据失败"
  }
  clearFiles()
})

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
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
