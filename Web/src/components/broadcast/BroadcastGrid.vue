<template>
  <div v-if="songsStore.loading" style="display:flex;justify-content:center;padding:48px;">
    <n-spin size="medium" />
  </div>

  <div v-else class="broadcast-grid">
    <n-card v-for="date in props.weekDates" :key="date" :title="dayLabel(date)" class="broadcast-day-card">
      <n-space vertical style="width:100%;">
        <div v-for="period in periods" :key="`${date}-${period.key}`">
          <n-space align="center" style="margin-bottom:8px;">
            <n-text strong :style="{ color: 'var(--theme-color)' }">{{ period.label }}</n-text>
            <n-button size="tiny" dashed @click="addSong(date, period.key)">+ {{ t("broadcast.addSong") }}</n-button>
          </n-space>

          <n-space v-for="song in songsFor(date, period.key)" :key="song.id" align="center" style="width:100%;">
            <div style="flex:1;min-width:0;">
              <n-input
                :value="titleDrafts[song.id] ?? song.title"
                size="small"
                @update:value="(v: string) => updateTitle(song.id, v)"
                @blur="saveTitle(song.id, titleDrafts[song.id] ?? song.title)"
              />
              <n-space v-if="!song.period" size="small">
                <n-text type="warning">历史时段待确认</n-text>
                <n-button size="tiny" @click="confirmPeriod(song.id, 'noon')">中午</n-button>
                <n-button size="tiny" @click="confirmPeriod(song.id, 'afternoon')">下午</n-button>
              </n-space>
              <div v-if="warningsFor(song).length" class="duplicate-warning">
                <div v-for="warning in warningsFor(song)" :key="warning.id">
                  {{ t("dorm.duplicateWarning", { date: warning.date, title: warning.title }) }}
                </div>
              </div>
            </div>
            <AudioPlayButton
              v-if="song.filePath"
              size="tiny"
              :file-path="song.filePath"
              :title="song.title"
            />
            <n-button size="tiny" type="error" @click="removeSong(song.id)">
              <template #icon>
                <Delete theme="outline" :size="14" :strokeWidth="3" />
              </template>
            </n-button>
          </n-space>
        </div>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSongDuplicateWarnings } from "../../composables/useSongDuplicateWarnings"
import { Delete } from "@icon-park/vue-next"
import AudioPlayButton from "../common/AudioPlayButton.vue"
import { dayjs } from "../../utils/datetime"
import type { Song } from "../../api/types"
import { queueTitle, flushTitle } from "../../utils/pendingSaves"
import { broadcastPeriod } from "../../utils/broadcastPeriod"

const props = defineProps<{
  weekDates: string[]
  columnMap: Record<string, string>
}>()

const { t, weekdayName } = useI18n()
const songsStore = useSongsStore()
const { findWarnings: duplicateWarnings } = useSongDuplicateWarnings("broadcast")
const titleDrafts = ref<Record<number, string>>({})

const periods = computed(() => [
  { key: "noon" as const, label: t("broadcast.noon") },
  { key: "afternoon" as const, label: t("broadcast.afternoon") },
])

const warningMap = computed(() => {
  const allSongs = [...songsStore.dormSongs, ...songsStore.broadcastSongs]
  return new Map(songsStore.broadcastSongs.map((song) => [song.id, duplicateWarnings(song, allSongs)]))
})

function dayLabel(date: string): string {
  const idx = dayIndexFromDate(date)
  const column = props.columnMap[String(idx)] || ""
  return `${date} ${weekdayName(idx)}${column && column !== "无" ? ` · ${column}` : ""}`
}

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function songsFor(date: string, period: "noon" | "afternoon"): Song[] {
  return songsStore.broadcastSongs
    .filter((s) => {
      if (s.date !== date) return false
      const p = broadcastPeriod(s)
      return p === period
    })
    .sort((a, b) => a.id - b.id)
}

async function saveTitle(id: number, title: string) {
  try {
    await flushTitle(id)
  } catch (err: any) {
    console.error(err)
  }
}

function updateTitle(id: number, title: string) {
  const song = songsStore.broadcastSongs.find(s => s.id === id)
  titleDrafts.value[id] = title
  queueTitle(id, title, async value => {
    await songsStore.updateSong(id, { title: value })
    if (titleDrafts.value[id] === value) delete titleDrafts.value[id]
  }, song ? { type: "broadcast", originalTitle: song.title, date: song.date, createdAt: song.createdAt } : undefined)
}

async function confirmPeriod(id: number, period: "noon" | "afternoon") {
  await flushTitle(id)
  await songsStore.updateSong(id, { period })
}

async function addSong(date: string, period: "noon" | "afternoon") {
  try {
    await songsStore.addSong({ type: "broadcast", date, period, title: "" })
  } catch (err: any) {
    console.error(err)
  }
}

async function removeSong(id: number) {
  try {
    await flushTitle(id)
    await songsStore.deleteSong(id)
  } catch (err: any) {
    console.error(err)
  }
}

function warningsFor(song: Song) {
  return warningMap.value.get(song.id) || []
}

onMounted(() => {
  songsStore.fetchSongs("broadcast")
})
</script>

<style scoped>
.broadcast-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(1, 1fr);
}

@media (min-width: 640px) {
  .broadcast-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1024px) {
  .broadcast-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.broadcast-day-card {
  min-width: 0;
}

.duplicate-warning {
  color: var(--color-warning);
  font-size: 12px;
  line-height: 1.4;
}
</style>
