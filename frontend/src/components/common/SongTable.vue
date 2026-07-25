<template>
  <div style="padding:16px;">
    <n-space align="center" style="margin-bottom:16px;" wrap>
      <n-h2 style="margin:0;">{{ title }}</n-h2>
      <n-button v-if="config.importXlsxEnabled" size="small" @click="handleImport">{{ t("broadcast.importXlsx") }}</n-button>
      <n-button v-if="config.exportXlsxEnabled" size="small" @click="handleExport">{{ t("broadcast.exportXlsx") }}</n-button>
    </n-space>

    <n-space align="center" style="margin-bottom:12px;">
      <n-button size="small" @click="prevWeek">← {{ t("export.lastWeek") }}</n-button>
      <n-text strong>{{ weekLabel }}</n-text>
      <n-button size="small" @click="nextWeek">{{ t("export.nextWeek") }} →</n-button>
      <n-button size="small" @click="goCurrentWeek">{{ t("export.thisWeek") }}</n-button>
    </n-space>

    <n-data-table :columns="columns" :data="editingSongs" :loading="songsStore.loading" :bordered="true" :max-height="480" />

    <n-space style="margin-top:12px;">
      <n-button dashed @click="addRow">+ {{ t("broadcast.addSong") }}</n-button>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { h, ref, onMounted, watch, computed } from "vue"
import { useI18n } from "../../i18n"
import { useSongsStore } from "../../stores/songs"
import { useSettingsStore } from "../../stores/settings"
import { useAudioPlayer } from "../../composables/useAudioPlayer"
import { useAppConfig } from "../../composables/useAppConfig"
import { importXlsx, exportXlsx, createSong, deleteSong, updateSong } from "../../api/songs"
import { useMessage } from "naive-ui"
import type { Song, SongType, TimeSlot } from "../../api/types"
import { NButton, NDatePicker, NInput, NSelect } from "naive-ui"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import isSameOrAfter from "dayjs/plugin/isSameOrAfter"
import isSameOrBefore from "dayjs/plugin/isSameOrBefore"

dayjs.extend(isoWeek)
dayjs.extend(isSameOrAfter)
dayjs.extend(isSameOrBefore)

const props = defineProps<{
  type: SongType
  title: string
  selectedDate?: string
}>()

const { t } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const message = useMessage()
const player = useAudioPlayer()
const { config } = useAppConfig()
const editingSongs = ref<Song[]>([])

const currentWeek = ref(dayjs(props.selectedDate || undefined).startOf("isoWeek"))

watch(() => props.selectedDate, (date) => {
  if (date) currentWeek.value = dayjs(date).startOf("isoWeek")
})

const sourceList = computed(() => props.type === "dorm" ? songsStore.dormSongs : songsStore.broadcastSongs)

const filteredSongs = computed(() => {
  const start = currentWeek.value.startOf("isoWeek")
  const end = currentWeek.value.endOf("isoWeek")
  return sourceList.value.filter(s => {
    const d = dayjs(s.date)
    const inWeek = d.isSameOrAfter(start, "day") && d.isSameOrBefore(end, "day")
    if (!inWeek) return false
    if (props.selectedDate) return s.date === props.selectedDate
    return true
  })
})

onMounted(() => { fetchData() })

watch(filteredSongs, (val) => {
  editingSongs.value = JSON.parse(JSON.stringify(val))
}, { immediate: true, deep: true })

async function fetchData() {
  try { await songsStore.fetchSongs(props.type) }
  catch { message.error("加载失败") }
}

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function slotsForDate(dateStr: string): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return settingsStore.timeSlots.filter(s => s.dayIndex === dayIdx).sort((a, b) => a.order - b.order)
}

const weekLabel = computed(() => {
  const year = currentWeek.value.isoWeekYear()
  const week = currentWeek.value.isoWeek()
  return `${year} 年第 ${week} 周`
})

function prevWeek() { currentWeek.value = currentWeek.value.add(-1, "week") }
function nextWeek() { currentWeek.value = currentWeek.value.add(1, "week") }
function goCurrentWeek() { currentWeek.value = dayjs().startOf("isoWeek") }

function updateLocal(id: number, data: Partial<Song>) {
  const idx = editingSongs.value.findIndex(s => s.id === id)
  if (idx >= 0) {
    editingSongs.value[idx] = { ...editingSongs.value[idx], ...data }
  }
}

async function saveSong(id: number, data: Partial<Song>) {
  try {
    await songsStore.updateSong(id, data)
  } catch (err: any) {
    const msg = err?.message || "保存失败"
    message.error(msg.includes("已存在歌曲") ? "该时段已有歌曲，请更换时段或先删除原歌曲" : msg)
  }
}

const baseColumns = [
  {
    title: t("broadcast.date"), key: "date", width: 160,
    render: (row: Song) => h(NDatePicker, {
      value: row.date ? dayjs(row.date).valueOf() : null,
      type: "date",
      onUpdateValue: (v: number | null) => {
        if (!v) return
        const date = dayjs(v).format("YYYY-MM-DD")
        updateLocal(row.id, { date })
      },
      onBlur: () => saveSong(row.id, { date: row.date }),
    }),
  },
  { title: t("broadcast.weekday"), key: "weekday", width: 80 },
]

const titleColumn = {
  title: t("broadcast.title"), key: "title", width: 240,
  render: (row: Song) => h(NInput, {
    value: row.title,
    placeholder: t("broadcast.title"),
    onUpdateValue: (v: string) => updateLocal(row.id, { title: v }),
    onBlur: () => saveSong(row.id, { title: row.title }),
  }),
}

const remarkColumn = {
  title: t("broadcast.remark"), key: "remark", width: 200,
  render: (row: Song) => h(NInput, {
    value: row.remark,
    placeholder: t("broadcast.remark"),
    onUpdateValue: (v: string) => updateLocal(row.id, { remark: v }),
    onBlur: () => saveSong(row.id, { remark: row.remark }),
  }),
}

const slotColumn = {
  title: t("advanced.timeSlots"), key: "timeSlotId", width: 140,
  render: (row: Song) => {
    const slots = slotsForDate(row.date)
    const options = slots.map(s => ({ label: s.time, value: s.id }))
    return h(NSelect, {
      value: row.timeSlotId,
      placeholder: t("songImport.slotPlaceholder"),
      options,
      clearable: true,
      onUpdateValue: (v: string | null) => saveSong(row.id, { timeSlotId: v }),
    })
  },
}

const actionsColumn = {
  title: t("broadcast.actions"), key: "actions", width: 150,
  render: (row: Song) => h("div", { style: "display:flex;gap:8px;" }, [
    row.filePath ? h(NButton, {
      size: "small",
      type: player.currentFile.value === row.filePath ? "primary" : "default",
      onClick: () => player.play(row.filePath, row.title),
    }, () => player.currentFile.value === row.filePath ? "暂停" : "试听") : null,
    h(NButton, {
      size: "small",
      type: "error",
      onClick: () => handleDelete(row.id),
    }, () => t("broadcast.delete")),
  ]),
}

const columns = computed(() => {
  const cols = [...baseColumns]
  if (props.type === "dorm") cols.push(slotColumn)
  cols.push(titleColumn, remarkColumn, actionsColumn)
  return cols
})

async function handleDelete(id: number) {
  try { await songsStore.deleteSong(id) }
  catch { message.error("删除失败") }
}

async function addRow() {
  try {
    const date = props.selectedDate || currentWeek.value.format("YYYY-MM-DD")
    await createSong({ type: props.type, date, title: "新歌曲" })
    await fetchData()
  } catch { message.error("添加失败") }
}

async function handleImport() {
  const i = document.createElement("input")
  i.type = "file"; i.accept = ".xlsx"
  i.onchange = async () => {
    const f = i.files?.[0]
    if (!f) return
    try {
      const r = await importXlsx(props.type, f)
      message.info(`导入 ${r.inserted} 首`)
      await fetchData()
    } catch { message.error("导入失败") }
  }
  i.click()
}

async function handleExport() {
  try {
    const b = await exportXlsx(props.type)
    const u = URL.createObjectURL(b)
    const a = document.createElement("a")
    a.href = u
    a.download = props.type === "dorm" ? "宿舍歌单.xlsx" : "播音歌单.xlsx"
    a.click()
    URL.revokeObjectURL(u)
  } catch { message.error("导出失败") }
}
</script>
