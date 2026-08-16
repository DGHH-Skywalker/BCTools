<template>
  <div v-if="showInitialLoading" style="display:flex;justify-content:center;padding:48px;">
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
              <DormSongRow
                :song="song"
                :slot-options="slotOptionsForDate(song.date)"
                :open="openSlotSongId === song.id"
                :warnings="duplicateWarnings(song)"
                @update-title="updateTitle"
                @save-title="saveTitle"
                @remove="removeSong"
                @assign-slot="assignSongToSlot"
                @toggle-popover="(id) => (openSlotSongId = id)"
              />
            </template>
            <template #footer>
              <div v-if="(slotLists[getSlotKey(date, sec.slotId)] || []).length === 0" class="empty-slot-placeholder">
              </div>
            </template>
          </draggable>
        </div>
      </div>
    </n-card>

    <n-card class="dorm-day-card dorm-pending-card">
      <template #header>
        <div class="day-title-block">
          <span class="day-title-text">{{ t("dorm.pendingCard") }}</span>
          <n-text depth="3" style="font-size:12px;">{{ t("dorm.pendingCardHint") }}</n-text>
        </div>
      </template>
      <draggable
        v-model="slotLists[PENDING_KEY]"
        item-key="id"
        handle=".drag-handle"
        :animation="150"
        :group="{ name: 'dorm-songs' }"
        ghost-class="song-row-ghost"
        drag-class="song-row-drag"
        class="slot-draggable"
        @change="onPendingChange"
      >
        <template #item="{ element: song }">
          <DormSongRow
            :song="song"
            :slot-options="slotOptionsForDate(song.date)"
            :open="openSlotSongId === song.id"
            :warnings="duplicateWarnings(song)"
            show-date
            @update-title="updateTitle"
            @save-title="saveTitle"
            @remove="removeSong"
            @assign-slot="assignSongToSlot"
            @toggle-popover="(id) => (openSlotSongId = id)"
          />
        </template>
        <template #footer>
          <n-empty
            v-if="(slotLists[PENDING_KEY] || []).length === 0"
            size="small"
            :description="t('dorm.pendingCardEmpty')"
            style="padding:12px 0;"
          />
        </template>
      </draggable>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, reactive } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import DormSongRow from "./DormSongRow.vue"
import { findSimilarSongs } from "../../utils/songSimilarity"
import draggable from "vuedraggable"
import { dayjs } from "../../utils/datetime"
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

const editingSongs = ref<Song[]>([])
const slotLists = reactive<Record<string, Song[]>>({})
const openSlotSongId = ref<number | null>(null)
const pendingTitles = reactive<Record<number, string>>({})
const hasLoadedOnce = ref(false)

// 只在首次加载时显示整页 spinner。添加/删除/拖拽歌曲后会重新拉取列表，
// 若此时也切换到 spinner，整个网格会闪一下，看起来像「刷新了页面」。
const showInitialLoading = computed(() => songsStore.loading && !hasLoadedOnce.value)

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
  // 只列出真实时段。未分配的歌曲统一收纳到第八个「待分配」卡片，
  // 不再在每天的卡片里重复出现一个未分配区。
  return slotsForDate(date).map((slot) => ({ label: slot.time, slotId: slot.id }))
}

// 第八个卡片：收纳本周（props.weekDates）内所有未分配时段的歌曲。
// 指向已被删除时段的歌曲也算未分配，否则它们会在界面上彻底消失。
const PENDING_KEY = "__pending__"

function pendingSongs(): Song[] {
  const dateSet = new Set(props.weekDates)
  return editingSongs.value
    .filter((s) => {
      if (!dateSet.has(s.date)) return false
      if (!s.timeSlotId) return true
      return !slotsForDate(s.date).some((sl) => sl.id === s.timeSlotId)
    })
    .sort((a, b) => a.date.localeCompare(b.date) || a.id - b.id)
}

function songsForSlot(date: string, slotId: string | null): Song[] {
  if (!slotId) return []
  return editingSongs.value
    .filter((s) => s.date === date && s.timeSlotId === slotId)
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
}

function sameSongOrder(a: Song[], b: Song[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    // 必须同时比较 id 与 timeSlotId：只比 id 会漏掉「歌曲仍在本列表、但时段已改」的情况，
    // 导致列表不重建、本地对象保留旧 timeSlotId，歌名旁的时段按钮显示成旧时段且点不动。
    if (a[i].id !== b[i].id) return false
    if ((a[i].timeSlotId ?? null) !== (b[i].timeSlotId ?? null)) return false
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
  const pending = pendingSongs()
  if (!(PENDING_KEY in slotLists) || !sameSongOrder(slotLists[PENDING_KEY], pending)) {
    slotLists[PENDING_KEY] = pending
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

async function assignSongToSlot(songId: number, slotId: string) {
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

// 拖入「待分配」卡片＝取消时段分配。日期保持不变，因为该卡片按日期混合展示，
// 没有单一目标日期可以套用。
async function onPendingChange(e: any) {
  if (!e.added) return
  const song = e.added.element as Song
  try {
    await songsStore.updateSong(song.id, { timeSlotId: null } as Partial<Song>)
  } catch (err: any) {
    console.error("移回待分配失败", err)
  }
}

onMounted(async () => {
  try {
    await songsStore.fetchSongs("dorm")
  } finally {
    hasLoadedOnce.value = true
  }
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

.dorm-pending-card {
  border: 1px dashed var(--color-warning);
}

.slot-draggable {
  min-height: 24px;
}

.empty-slot-placeholder {
  min-height: 24px;
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
