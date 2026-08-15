<template>
  <div v-if="songsStore.loading" style="display:flex;justify-content:center;padding:48px;">
    <n-spin size="medium" />
  </div>

  <div v-else class="dorm-grid">
    <n-card v-for="date in props.weekDates" :key="date" class="dorm-day-card">
      <template #header>
        <div class="day-title-block">
          <span class="day-title-text">{{ dayLabel(date) }}</span>
          <n-button size="tiny" dashed @click="goToImport(date)">+ {{ t("broadcast.addSong") }}</n-button>
        </div>
      </template>
      <div class="day-sections">
        <div
          v-for="sec in sectionsForDate(date)"
          :key="sec.slotId || '__unassigned__'"
          class="slot-section"
          :class="{ 'slot-section-unassigned': !sec.slotId }"
        >
          <div class="slot-section-title">{{ sec.label }}</div>
          <draggable
            v-model="slotLists[getSlotKey(date, sec.slotId)]"
            item-key="id"
            handle=".drag-handle"
            :animation="150"
            :group="{ name: 'dorm-songs' }"
            ghost-class="song-row-ghost"
            drag-class="song-row-drag"
            class="slot-draggable"
            @change="(e: any) => onSlotChange(date, sec.slotId, e)"
          >
              <template #item="{ element: song }">
                <div class="song-row">
                  <span class="drag-handle" :title="t('dorm.dragSort')">
                    <svg viewBox="0 0 10 16" width="10" height="16" aria-hidden="true"
                    >
                      <circle cx="2.5" cy="3" r="1.3" fill="currentColor" />
                      <circle cx="2.5" cy="8" r="1.3" fill="currentColor" />
                      <circle cx="2.5" cy="13" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="3" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="8" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="13" r="1.3" fill="currentColor" />
                    </svg>
                  </span>
                  <div class="song-info">
                    <n-input
                      class="no-drag"
                      :value="song.title"
                      size="small"
                      :placeholder="t('broadcast.songTitle')"
                      @update:value="(v: string) => updateTitle(song.id, v)"
                      @blur="saveTitle(song.id)"
                    />
                    <div v-if="duplicateWarnings(song).length" class="duplicate-warning"
                    >
                      <div v-for="w in duplicateWarnings(song)" :key="w.id">
                        {{ t("dorm.duplicateWarning", { date: w.date, title: w.title }) }}
                      </div>
                    </div>
                  </div>

                  <n-popover
                    class="no-drag"
                    trigger="click"
                    placement="bottom"
                    :show-arrow="false"
                    :show="openSlotSongId === song.id"
                    @clickoutside="openSlotSongId = null"
                  >
                    <template #trigger>
                      <n-button
                        class="no-drag"
                        size="tiny"
                        :title="t('dorm.assignSlot')"
                        @click="openSlotSongId = song.id"
                      >
                        {{ slotLabelForSong(song) }}
                      </n-button>
                    </template>
                    <div class="slot-option-list"
                    >
                      <div
                        v-for="opt in slotOptionsForDate(song.date)"
                        :key="opt.value"
                        class="slot-option"
                        :class="{ active: opt.value === song.timeSlotId }"
                        @click="assignSongToSlot(song.id, opt.value as string | null)"
                      >{{ opt.label }}</div>
                    </div>
                  </n-popover>

                  <AudioPlayButton
                    v-if="song.filePath"
                    class="no-drag"
                    size="tiny"
                    :file-path="song.filePath"
                    :title="song.title"
                  />
                  <n-button class="no-drag" size="tiny" type="error" @click="removeSong(song.id)"
                  >
                    <template #icon>
                      <Delete theme="outline" :size="14" :strokeWidth="3" />
                    </template>
                  </n-button>
                </div>
              </template>
              <template #footer>
                <div v-if="(slotLists[getSlotKey(date, sec.slotId)] || []).length === 0" class="empty-slot-placeholder">
                </div>
              </template>
            </draggable>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, reactive } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import { useAudioPlayer } from "../../composables/useAudioPlayer"
import AudioPlayButton from "../common/AudioPlayButton.vue"
import { Delete } from "@icon-park/vue-next"
import { findSimilarSongs } from "../../utils/songSimilarity"
import draggable from "vuedraggable"
import dayjs from "dayjs"
import type { Song, TimeSlot } from "../../api/types"

const props = defineProps<{
  weekDates: string[]
}>()

const emit = defineEmits<{
  (e: "open-import", payload: { date: string; timeSlotId?: string }): void
}>()

const { t, weekdayName } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const player = useAudioPlayer()

const editingSongs = ref<Song[]>([])
const slotLists = reactive<Record<string, Song[]>>({})
const openSlotSongId = ref<number | null>(null)
const pendingTitles = reactive<Record<number, string>>({})

function getSlotKey(date: string, slotId: string | null): string {
  return `${date}|${slotId || "__unassigned__"}`
}

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function dayLabel(date: string): string {
  const idx = dayIndexFromDate(date)
  return `${date} ${weekdayName(idx)}`
}

function slotsForDate(dateStr: string): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return settingsStore.timeSlots
    .filter((s) => s.dayIndex === dayIdx)
    .sort((a, b) => parseTime(a.time) - parseTime(b.time))
}

interface DaySection {
  label: string
  slotId: string | null
}

function sectionsForDate(date: string): DaySection[] {
  const slots = slotsForDate(date)
  const sections: DaySection[] = [{ label: t("dorm.unassigned"), slotId: null }]
  for (const slot of slots) {
    sections.push({ label: slot.time, slotId: slot.id })
  }
  return sections
}

function songsForSlot(date: string, slotId: string | null): Song[] {
  const slots = slotsForDate(date)
  if (slotId) {
    return editingSongs.value
      .filter((s) => s.date === date && s.timeSlotId === slotId)
      .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
  }
  return editingSongs.value
    .filter((s) => s.date === date && (!s.timeSlotId || !slots.some((sl) => sl.id === s.timeSlotId)))
    .sort((a, b) => a.id - b.id)
}

function sameSongOrder(a: Song[], b: Song[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i].id !== b[i].id) return false
  }
  return true
}

function rebuildSlotLists() {
  editingSongs.value = JSON.parse(JSON.stringify(songsStore.dormSongs))
  // 仅在结构（id 顺序/数量）变化时重建 slotLists，避免输入标题时丢失焦点；
  // 注意 key 不存在时必须无条件初始化，否则空时段的列表恒为 undefined，
  // 拖拽放入时 vuedraggable 展开 undefined 会抛错，导致歌曲卡片“被吞掉”
  for (const date of props.weekDates) {
    for (const sec of sectionsForDate(date)) {
      const key = getSlotKey(date, sec.slotId)
      const next = songsForSlot(date, sec.slotId)
      if (!(key in slotLists) || !sameSongOrder(slotLists[key], next)) {
        slotLists[key] = next
      }
    }
  }
}

watch(
  [() => props.weekDates, () => songsStore.dormSongs, () => settingsStore.timeSlots],
  rebuildSlotLists,
  { deep: true, immediate: true },
)

function duplicateWarnings(song: Song) {
  if (!song.title) return []
  return findSimilarSongs(song.title, song.date, songsStore.dormSongs, settingsStore.duplicateCheckDays || 30)
}

function parseTime(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return new Date(1970, 0, 1, h || 0, m || 0).getTime()
}

function updateTitle(id: number, title: string) {
  pendingTitles[id] = title
  const song = editingSongs.value.find((s) => s.id === id)
  if (song) {
    song.title = title
  }
  for (const key in slotLists) {
    const found = slotLists[key].find((s) => s.id === id)
    if (found) {
      found.title = title
    }
  }
}

async function saveTitle(id: number) {
  const title = pendingTitles[id]
  if (title === undefined) return
  delete pendingTitles[id]
  try {
    await songsStore.updateSong(id, { title })
  } catch (err: any) {
    console.error(err)
  }
}

function goToImport(date: string, timeSlotId?: string) {
  emit("open-import", { date, timeSlotId })
}

async function removeSong(id: number) {
  try {
    await songsStore.deleteSong(id)
  } catch (err: any) {
    console.error(err)
  }
}

function slotOptionsForDate(date: string) {
  const slots = slotsForDate(date)
  const options = slots.map((slot) => ({ label: slot.time, value: slot.id }))
  options.unshift({ label: t("dorm.unassigned"), value: "__unassigned__" })
  return options
}

function slotLabelForSong(song: Song): string {
  if (!song.timeSlotId) return t("dorm.unassigned")
  const slot = settingsStore.timeSlots.find((s) => s.id === song.timeSlotId)
  return slot ? slot.time : t("dorm.unassigned")
}

async function assignSongToSlot(songId: number, slotId: string | null) {
  openSlotSongId.value = null
  const realSlotId = slotId === "__unassigned__" ? null : slotId
  try {
    await songsStore.updateSong(songId, { timeSlotId: realSlotId })
  } catch (err: any) {
    console.error("分配时段失败", err)
  }
}

async function onSlotChange(date: string, slotId: string | null, e: any) {
  const key = getSlotKey(date, slotId)
  const list = slotLists[key]

  if (e.moved) {
    try {
      await songsStore.reorderSongs(list.map((s) => s.id))
    } catch (err: any) {
      console.error("排序保存失败", err)
    }
  } else if (e.added) {
    const song = e.added.element as Song
    try {
      await songsStore.updateSong(song.id, { date, timeSlotId: slotId })
      await songsStore.reorderSongs(list.map((s) => s.id))
    } catch (err: any) {
      console.error("跨卡片拖拽保存失败", err)
    }
  }
}

onMounted(() => {
  songsStore.fetchSongs("dorm")
})
</script>

<style scoped>
.dorm-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(1, 1fr);
}

@media (min-width: 640px) {
  .dorm-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1200px) {
  .dorm-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.dorm-day-card {
  min-width: 0;
}

.day-title-block {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.day-title-text {
  font-size: 16px;
  line-height: 1.4;
}

.day-sections {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.slot-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.slot-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  padding-left: 2px;
}

.slot-section-unassigned .slot-section-title {
  color: var(--color-warning);
}

.song-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.drag-handle {
  cursor: grab;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}

.song-info {
  flex: 1;
  min-width: 0;
}

.duplicate-warning {
  color: var(--color-warning);
  font-size: 12px;
  line-height: 1.4;
}

.slot-draggable {
  min-height: 24px;
}

.empty-slot-placeholder {
  min-height: 24px;
}

.time-option-list,
.slot-option-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 80px;
}

.slot-option {
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}

.slot-option:hover {
  background-color: var(--color-bg-light);
}

.slot-option.active {
  background-color: var(--color-primary-light);
  color: var(--color-primary);
}

.song-row-ghost {
  opacity: 0.5;
  background-color: var(--color-bg-light);
}

.song-row-drag {
  background-color: var(--color-bg);
  box-shadow: var(--shadow-md);
}
</style>
