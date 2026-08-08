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
      <n-timeline size="medium">
        <n-timeline-item
          v-for="sec in sectionsForDate(date)"
          :key="sec.slotId || '__unassigned__'"
          :type="sec.slotId ? 'default' : 'warning'"
        >
          <template #footer>
            <n-popover
              v-if="sec.slotId"
              trigger="manual"
              placement="bottom"
              :show-arrow="false"
              :show="openSlotId === sec.slotId"
            >
              <template #trigger>
                <span
                  class="slot-time-edit"
                  :title="t('dorm.editSlotTime')"
                  @click="toggleSlotEdit(sec.slotId!)"
                >{{ sec.label }}</span>
              </template>
              <div class="time-option-list">
                <div
                  v-for="opt in presetOptions"
                  :key="opt.value"
                  class="time-option"
                  :class="{ active: opt.value === sec.label }"
                  @click="onSlotTimePicked(sec.slotId!, opt.value)"
                >{{ opt.label }}</div>
              </div>
            </n-popover>
            <span v-else class="slot-time-static">{{ sec.label }}</span>
          </template>

          <n-space vertical style="width:100%;">
            <n-text v-if="slotLists[getSlotKey(date, sec.slotId)].length === 0" depth="3">{{ t("common.empty") }}</n-text>

            <draggable
              v-else
              v-model="slotLists[getSlotKey(date, sec.slotId)]"
              item-key="id"
              handle=".drag-handle"
              :animation="150"
              :group="{ name: 'dorm-songs' }"
              @change="(e: any) => onSlotChange(date, sec.slotId, e)"
            >
              <template #item="{ element: song }">
                <div class="song-row">
                  <span class="drag-handle" :title="t('dorm.dragSort')">
                    <svg viewBox="0 0 10 16" width="10" height="16" aria-hidden="true">
                      <circle cx="2.5" cy="3" r="1.3" fill="currentColor" /><circle cx="2.5" cy="8" r="1.3" fill="currentColor" /><circle cx="2.5" cy="13" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="3" r="1.3" fill="currentColor" /><circle cx="7" cy="8" r="1.3" fill="currentColor" /><circle cx="7" cy="13" r="1.3" fill="currentColor" />
                    </svg>
                  </span>
                  <div class="song-info">
                    <n-input
                      :value="song.title"
                      size="small"
                      :placeholder="t('broadcast.songTitle')"
                      @update:value="(v: string) => updateTitle(song.id, v)"
                      @blur="saveTitle(song.id, song.title)"
                    />
                    <div v-if="duplicateWarnings(song).length" style="color:#d4a017;font-size:12px;line-height:1.4;">
                      <div v-for="w in duplicateWarnings(song)" :key="w.id">
                        {{ t("dorm.duplicateWarning", { date: w.date, title: w.title }) }}
                      </div>
                    </div>
                  </div>
                  <n-button
                    v-if="song.filePath"
                    size="tiny"
                    :type="player.currentFile.value === song.filePath ? 'primary' : 'default'"
                    @click="player.play(song.filePath, song.title)"
                  >
                    {{ player.currentFile.value === song.filePath ? t('common.pause') : t('common.listen') }}
                  </n-button>
                  <n-button size="tiny" type="error" @click="removeSong(song.id)">
                    <template #icon>
                      <Delete theme="outline" :size="14" :strokeWidth="3" />
                    </template>
                  </n-button>
                </div>
              </template>
            </draggable>
          </n-space>
        </n-timeline-item>
      </n-timeline>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, reactive } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import { useAudioPlayer } from "../../composables/useAudioPlayer"
import { Delete } from "@icon-park/vue-next"
import { findSimilarSongs } from "../../utils/songSimilarity"
import draggable from "vuedraggable"
import { useRouter } from "vue-router"
import dayjs from "dayjs"
import type { Song, TimeSlot } from "../../api/types"

const props = defineProps<{
  weekDates: string[]
}>()

const { t, weekdayName } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const player = useAudioPlayer()
const router = useRouter()

const slotLists = reactive<Record<string, Song[]>>({})

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
  return settingsStore.timeSlots.filter((s) => s.dayIndex === dayIdx).sort((a, b) => parseTime(a.time) - parseTime(b.time))
}

interface DaySection {
  label: string
  slotId: string | null
  canAdd: boolean
}

// 把某一天的时段配置展开为时段列表；未配置时段的歌曲归入「未分配时段」
function sectionsForDate(date: string): DaySection[] {
  const slots = slotsForDate(date)
  const sections: DaySection[] = slots.map((slot) => ({
    label: slot.time,
    slotId: slot.id,
    canAdd: true,
  }))
  sections.push({ label: t("dorm.unassigned"), slotId: null, canAdd: false })
  return sections
}

function songsForSlot(date: string, slotId: string | null): Song[] {
  const slots = slotsForDate(date)
  const daySongs = songsStore.dormSongs.filter((s) => s.date === date)
  if (slotId) {
    return daySongs
      .filter((s) => s.timeSlotId === slotId)
      .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.id - b.id)
  }
  return daySongs
    .filter((s) => !s.timeSlotId || !slots.some((sl) => sl.id === s.timeSlotId))
    .sort((a, b) => a.id - b.id)
}

function refreshSlotLists() {
  for (const date of props.weekDates) {
    for (const sec of sectionsForDate(date)) {
      const key = getSlotKey(date, sec.slotId)
      slotLists[key] = songsForSlot(date, sec.slotId)
    }
  }
}

watch(
  () => [props.weekDates, songsStore.dormSongs, settingsStore.timeSlots],
  refreshSlotLists,
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

// 预设时间段：取自 settingsStore.timeSlots 中已配置的去重时间，按时分排序
const presetOptions = computed(() => {
  const times = Array.from(new Set(settingsStore.timeSlots.map((s) => s.time).filter(Boolean)))
  times.sort((a, b) => parseTime(a) - parseTime(b))
  return times.map((time) => ({ label: time, value: time }))
})

const openSlotId = ref<string | null>(null)

function toggleSlotEdit(slotId: string) {
  openSlotId.value = openSlotId.value === slotId ? null : slotId
}

async function onSlotTimePicked(slotId: string, newTime: string | null) {
  openSlotId.value = null
  if (!newTime) return
  const slot = settingsStore.timeSlots.find((s) => s.id === slotId)
  if (!slot || slot.time === newTime) return
  const sorted = settingsStore.timeSlots
    .map((s) => (s.id === slotId ? { ...s, time: newTime } : s))
    .sort((a, b) => (a.dayIndex !== b.dayIndex ? a.dayIndex - b.dayIndex : a.order - b.order))
  try {
    await settingsStore.updateSettings({ timeSlots: sorted } as any)
  } catch (err: any) {
    console.error("调整时段时间失败", err)
  }
}

function updateTitle(id: number, title: string) {
  for (const key in slotLists) {
    const idx = slotLists[key].findIndex((s) => s.id === id)
    if (idx >= 0) {
      slotLists[key][idx] = { ...slotLists[key][idx], title }
      break
    }
  }
}

async function saveTitle(id: number, title: string) {
  try {
    await songsStore.updateSong(id, { title })
  } catch (err: any) {
    console.error(err)
  }
}

function goToImport(date: string) {
  router?.push({ path: "/song/import", query: { date } })
}

async function removeSong(id: number) {
  try {
    await songsStore.deleteSong(id)
  } catch (err: any) {
    console.error(err)
  }
}

async function onSlotChange(date: string, slotId: string | null, e: any) {
  const key = getSlotKey(date, slotId)
  const list = slotLists[key]

  if (e.moved) {
    // 同列表内排序
    try {
      await songsStore.reorderSongs(list.map((s) => s.id))
    } catch (err: any) {
      console.error("排序保存失败", err)
    }
  } else if (e.added) {
    // 跨卡片/跨时段拖入：更新歌曲归属日期与时段，并按目标列表顺序重排
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

.song-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.song-row + .song-row {
  margin-top: 8px;
}

.song-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.drag-handle {
  flex-shrink: 0;
  cursor: grab;
  color: var(--theme-color);
  opacity: 0.55;
  display: inline-flex;
  align-items: center;
  padding: 2px;
  user-select: none;
  transition: opacity 0.15s;
}

.drag-handle:hover {
  opacity: 1;
}

.drag-handle:active {
  cursor: grabbing;
}

.slot-time-edit,
.slot-time-static {
  color: var(--theme-color);
}

.slot-time-edit {
  cursor: pointer;
}

.slot-time-edit:hover {
  text-decoration: underline dotted;
}

.time-option-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 96px;
  max-height: 240px;
  overflow-y: auto;
  padding: 4px;
}

.time-option {
  padding: 4px 16px;
  border-radius: 4px;
  cursor: pointer;
  text-align: center;
  font-size: 14px;
  line-height: 1.4;
  color: #333;
  user-select: none;
  transition: background 0.15s;
}

.time-option:hover {
  background: var(--theme-color-light);
}

.time-option.active {
  background: var(--theme-color);
  color: #fff;
  font-weight: 500;
}

/* 时间轴线条与节点圆点统一为主题色 */
.dorm-day-card :deep(.n-timeline-item-timeline__line) {
  background-color: var(--theme-color);
}

.dorm-day-card :deep(.n-timeline-item--default-type .n-timeline-item-timeline__circle) {
  border-color: var(--theme-color);
}
</style>
