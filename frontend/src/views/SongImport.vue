<template>
  <div style="padding:16px;max-width:1200px;margin:0 auto;">
    <n-space align="center" style="margin-bottom:16px;">
      <n-button text @click="showCalendar=true" v-if="!showCalendar">
        ← {{ t("songImport.backToCalendar") }}
      </n-button>
      <n-h2 style="margin:0;">{{ pageTitle }}</n-h2>
    </n-space>

    <template v-if="showCalendar">
      <CalendarGrid
        :year="year"
        :month="month"
        :songs-map="songsMap"
        selection-mode="single"
        @date-click="onDateClick"
        @update:year="year = $event"
        @update:month="month = $event"
      />
    </template>

    <template v-else>
      <n-alert v-if="error" type="error" closable @close="error=''" style="margin-bottom:12px;">{{ error }}</n-alert>
      <n-alert v-if="noSlots" type="warning" style="margin-bottom:12px;">{{ t("songImport.noSlots") }}</n-alert>

      <n-card v-if="existingSongs.length > 0" style="margin-bottom:16px;" :title="existingTitle">
        <n-list>
          <n-list-item v-for="song in existingSongs" :key="song.id">
            <n-space align="center" justify="space-between" style="width:100%;" wrap>
              <n-space align="center">
                <n-text strong>{{ song.title }}</n-text>
                <n-tag v-if="song.timeSlotId" size="tiny" type="success">{{ slotName(song.timeSlotId) }}</n-tag>
              </n-space>
              <n-button v-if="song.filePath" size="tiny" @click="previewSong(song)">{{ t('common.listen') }}</n-button>
            </n-space>
          </n-list-item>
        </n-list>
      </n-card>

      <n-card style="margin-bottom:16px;" :title="t('songImport.uploadTitle')">
        <n-upload :default-upload="false" accept=".ncm,.mp3,.flac,.wav,.mp4" multiple @change="handleFiles">
          <n-upload-dragger>
            <div style="padding:24px;text-align:center;">
              <n-h3>{{ t("songImport.dropFiles") }}</n-h3>
              <n-p depth="3">{{ t("songImport.formatHint") }}</n-p>
            </div>
          </n-upload-dragger>
        </n-upload>
      </n-card>

      <n-p v-if="processingFiles.length > 0">
        {{ t("songImport.progress", { done: doneCount, total: processingFiles.length }) }}
      </n-p>

      <n-grid :cols="3" :x-gap="8" :y-gap="8" v-if="processingFiles.length > 0">
        <n-grid-item v-for="file in processingFiles" :key="file.id">
          <n-card :class="['conv-card', file.status]" size="small">
            <n-space vertical>
              <n-space align="center">
                <n-tag :type="statusTagType(file.status)" size="small">{{ file.fileType }}</n-tag>
                <n-ellipsis style="max-width:120px;">{{ file.fileName }}</n-ellipsis>
              </n-space>
              <n-spin v-if="file.status==='converting'" size="small">
                <template #description>处理中...</template>
              </n-spin>
              <n-space v-if="file.status==='done'" vertical>
                <n-space align="center">
                  <n-input :value="file.songTitle" size="small" style="width:200px;" @update:value="(v: string) => file.songTitle=v" @blur="saveMetadata(file)" :placeholder="t('songImport.editTitle')" />
                  <n-select
                    v-if="file.songId"
                    size="small"
                    :value="file.timeSlotId"
                    :placeholder="t('songImport.slotPlaceholder')"
                    :options="slotOptions"
                    :render-label="renderSlotLabel"
                    @update:value="(v: string | null) => assignSlot(file, v as string)"
                    style="width:100%;"
                  />
                  <n-text v-if="file.timeSlotId" type="success" depth="3">{{ t("songImport.assigned") }}</n-text>
                </n-space>
              </n-space>
              <n-text v-if="file.status==='error'" type="error" depth="3">{{ file.error }}</n-text>
              <n-button v-if="file.status==='error'" size="small" @click="retryFile(file)">{{ t("songImport.retry") }}</n-button>
            </n-space>
          </n-card>
        </n-grid-item>
      </n-grid>

      <n-empty v-if="processingFiles.length === 0" :description="t('songImport.empty')" style="margin-top:40px;" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive, h } from "vue"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { processFile, stashFile } from "../api/files"
import { createSong, updateSong } from "../api/songs"
import CalendarGrid from "../components/calendar/CalendarGrid.vue"
import { useAudioPlayer } from "../composables/useAudioPlayer"
import dayjs from "dayjs"
import type { TimeSlot } from "../api/types"
import type { SelectOption } from "naive-ui"

interface ProcessingFile {
  id: string
  file: File
  fileName: string
  fileType: string
  status: "pending" | "converting" | "done" | "error"
  songTitle: string
  tempFileName: string | null
  songId: number | null
  timeSlotId: string | null
  error: string
}

const { t } = useI18n()
const player = useAudioPlayer()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const error = ref("")
const showCalendar = ref(true)
const year = ref(dayjs().year())
const month = ref(dayjs().month() + 1)
const selectedDate = ref("")
const processingFiles = reactive<ProcessingFile[]>([])
const concurrency = 3
let running = 0

const pageTitle = computed(() => {
  if (showCalendar.value) return t("songImport.selectDate")
  return `${t("songImport.title")} - ${selectedDate.value}`
})

const doneCount = computed(() => processingFiles.filter(f => f.status === "done" || f.status === "error").length)
const noSlots = computed(() => {
  if (!selectedDate.value) return false
  const dayIdx = dayIndexFromDate(selectedDate.value)
  return !settingsStore.timeSlots.some(s => s.dayIndex === dayIdx)
})

const selectedDayIndex = computed(() => selectedDate.value ? dayIndexFromDate(selectedDate.value) : null)

const availableSlots = computed<TimeSlot[]>(() => {
  if (!selectedDayIndex.value) return []
  return settingsStore.timeSlots
    .filter(s => s.dayIndex === selectedDayIndex.value)
    .sort((a, b) => a.order - b.order)
})

const slotOptions = computed<SelectOption[]>(() => {
  return availableSlots.value.map(s => ({
    label: s.time,
    value: s.id,
  }))
})

const existingTitle = computed(() => `当日已点歌曲 (${existingSongs.value.length})`)
const existingSongs = computed(() => {
  if (!selectedDate.value) return []
  return songsStore.dormSongs.filter(s => s.date === selectedDate.value)
})

const songsMap = computed(() => {
  const map: Record<string, { totalSlots: number; filledSlots: number }> = {}
  for (const song of songsStore.dormSongs) {
    if (!map[song.date]) {
      const dayIdx = dayIndexFromDate(song.date)
      const total = settingsStore.timeSlots.filter(s => s.dayIndex === dayIdx).length
      map[song.date] = { totalSlots: total, filledSlots: 0 }
    }
    map[song.date].filledSlots++
  }
  return map
})

function slotName(slotId: string | null) {
  if (!slotId) return ""
  const slot = settingsStore.timeSlots.find(s => s.id === slotId)
  return slot ? slot.time : ""
}

function renderSlotLabel(option: SelectOption) {
  return option.label as string
}

function previewSong(song: { filePath: string; title: string }) {
  player.play(song.filePath, song.title)
}

onMounted(async () => {
  try {
    await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
  } catch {
    error.value = "加载数据失败"
  }
})

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function onDateClick(date: string) {
  if (!date) return
  selectedDate.value = date
  showCalendar.value = false
}

function extractFiles(fileList: any[]): File[] {
  const files: File[] = []
  for (const f of fileList) {
    if (f instanceof File) {
      files.push(f)
    } else if (f && f.file instanceof File) {
      files.push(f.file)
    } else if (f && f.raw instanceof File) {
      files.push(f.raw)
    }
  }
  return files
}

function statusTagType(status: ProcessingFile["status"]) {
  if (status === "done") return "success"
  if (status === "error") return "error"
  return "warning"
}

function handleFiles({ fileList }: any) {
  const files = extractFiles(fileList || [])
  for (const file of files) {
    const ext = file.name.split(".").pop()?.toLowerCase() || ""
    const item: ProcessingFile = {
      id: `f-${Date.now()}-${Math.random().toString(36).slice(2)}`,
      fileName: file.name,
      fileType: ext,
      file,
      status: "pending",
      songTitle: file.name.replace(/\.[^.]+$/, ""),
      tempFileName: null,
      songId: null,
      timeSlotId: null,
      error: "",
    }
    processingFiles.push(item)
  }
  runQueue()
}

async function runQueue() {
  if (running >= concurrency) return
  const next = processingFiles.find(f => f.status === "pending")
  if (!next) return
  running++
  await processItem(next)
  running--
  runQueue()
}

async function processItem(item: ProcessingFile) {
  item.status = "converting"
  item.error = ""
  const ext = item.fileType
  try {
    const result = ext === "mp3" ? await stashFile(item.file) : await processFile(item.file)
    item.tempFileName = result.tempFileName
    item.songTitle = result.title || item.songTitle

    if (selectedDate.value && item.songTitle) {
      try {
        const song = await createSong({
          type: "dorm",
          date: selectedDate.value,
          title: item.songTitle,
          filePath: result.tempFileName,
        })
        item.songId = song.id
        songsStore.dormSongs.push(song)
      } catch (err: any) {
        item.status = "error"
        item.error = err?.message || "创建歌曲失败"
        return
      }
    }
    item.status = "done"
  } catch (err: any) {
    item.status = "error"
    item.error = err?.message || "处理失败"
    console.error("File processing error:", err)
  }
}

async function saveMetadata(item: ProcessingFile) {
  if (!item.songId) return
  try {
    await updateSong(item.songId, { title: item.songTitle })
  } catch (err: any) {
    console.error("保存元数据失败", err)
  }
}

async function assignSlot(item: ProcessingFile, slotId: string) {
  if (!item.songId) return
  try {
    await songsStore.assignSong(item.songId, slotId)
    item.timeSlotId = slotId
  } catch (err: any) {
    error.value = err?.message?.includes("已存在歌曲") ? "该时段已有歌曲，请更换时段或先删除原歌曲" : (err?.message || "分配时段失败")
  }
}

function retryFile(item: ProcessingFile) {
  item.status = "pending"
  item.error = ""
  runQueue()
}
</script>

<style scoped>
.conv-card { margin-bottom:4px; }
.conv-card.error { border-color: #e74c3c; }
</style>
