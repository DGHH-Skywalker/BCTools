<template>
  <div style="padding:8px;">
    <n-alert v-if="error" type="error" closable @close="error = ''" style="margin-bottom: 12px;">{{ error }}</n-alert>
    <n-alert v-if="noSlots" type="warning" style="margin-bottom: 12px;">{{ t("songImport.noSlots") }}</n-alert>

    <ImportUploadZone
      :title="t('songImport.uploadTitle')"
      accept=".ncm,.mp3,.mp4,.m4a,.flac,.wav,.aac,.ogg,.wma,.ape"
      formats="支持 ncm / mp3 / m4a / flac / wav / aac / ogg / wma / ape / mp4"
      @change="handleFiles"
    />

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

    <n-empty v-if="processingFiles.length === 0" description="" style="margin-top: 40px;" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import { useFileProcessor } from "../../composables/useFileProcessor"
import { fetchStageMeta, markStageImported } from "../../api/decrypt"
import ImportUploadZone from "../song-import/ImportUploadZone.vue"
import ImportProcessingGrid from "../song-import/ImportProcessingGrid.vue"
import { dayjs } from "../../utils/datetime"
import type { SelectOption } from "naive-ui"

const props = defineProps<{
  date: string
  defaultSlotId?: string
  stageId?: string
}>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "song-added"): void
}>()

const { t } = useI18n()
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
  handleStagedFile,
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

function renderSlotLabel(option: SelectOption) {
  return option.label as string
}

onMounted(async () => {
  try {
    await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
  } catch {
    error.value = "加载数据失败"
  }
  clearFiles()
  if (props.stageId) {
    await importFromStage(props.stageId)
  }
})

// 从后端暂存区导入 um-react 解密后的音频。
// 音频已经在后端，因此只传 stageId 让后端就地入库——不再下载回浏览器再上传，
// 省掉一首歌两趟共约 24 MB 的本机传输。
// 成功后置 imported 标志，um-react 侧轮询到即删除对应卡片。
async function importFromStage(stageId: string) {
  try {
    const meta = await fetchStageMeta(stageId)
    const results = await handleStagedFile(stageId, meta.filename)
    if (results.some((r) => r.status === "done")) {
      await markStageImported(stageId)
      emit("song-added")
    } else {
      error.value = results[0]?.error || t("songImport.importDecryptedExpired")
    }
  } catch {
    error.value = t("songImport.importDecryptedExpired")
  }
}

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
</script>
