<template>
  <div style="padding:16px;max-width:1000px;margin:0 auto;">
    <n-h2>{{ t("organize.title") }}</n-h2>

    <n-card style="margin-bottom:16px;">
      <n-space vertical>
        <n-space align="center">
          <n-text>{{ t("organize.targetDir") }}:</n-text>
          <n-input v-model:value="targetDir" :placeholder="'SD卡目录，如 E:\\'" style="width:300px" />
          <n-button @click="browse">{{ t("organize.browse") }}</n-button>
        </n-space>
        <n-space align="center">
          <n-text>{{ t("organize.dateRange") }}:</n-text>
          <n-date-picker v-model:formatted-value="startDate" value-format="yyyy-MM-dd" type="date" />
          <n-text>~</n-text>
          <n-date-picker v-model:formatted-value="endDate" value-format="yyyy-MM-dd" type="date" />
        </n-space>
        <n-space align="center">
          <n-text>{{ t("organize.mode") }}:</n-text>
          <n-radio-group v-model:value="mode">
            <n-radio value="copy">{{ t("organize.modeCopy") }}</n-radio>
            <n-radio value="move">{{ t("organize.modeMove") }}</n-radio>
          </n-radio-group>
        </n-space>
        <n-button type="primary" @click="generateEntries">{{ t("organize.generate") }}</n-button>
      </n-space>
    </n-card>

    <n-card v-if="entries.length > 0" style="margin-bottom:16px;" :title="t('organize.title')">
      <n-data-table :columns="columns" :data="entries" :bordered="true" :max-height="400" />
      <n-space style="margin-top:12px;">
        <n-button type="primary" @click="() => doOrganize()" :loading="organizing">{{ t("organize.organize") }}</n-button>
      </n-space>
    </n-card>

    <n-empty v-else :description="t('organize.noEntries')" />

    <n-modal v-model:show="showConfirm" preset="dialog" :title="t('organize.confirmOverwrite')" positive-text="确认覆盖" negative-text="取消" @positive-click="() => doOrganize(true)">
      <n-ul>
        <n-li v-for="f in existingFiles" :key="f">{{ f }}</n-li>
      </n-ul>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from "vue"
import { useI18n } from "../i18n"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { organizeFiles, selectDir } from "../api/files"
import { useMessage } from "naive-ui"
import { NInput } from "naive-ui"
import dayjs from "dayjs"
import type { TimeSlot } from "../api/types"

interface OrganizeEntry {
  key: string
  date: string
  slot: TimeSlot
  source: string
  targetName: string
  songTitle: string
}

const { t } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const message = useMessage()

const targetDir = ref("")
const startDate = ref(dayjs().startOf("isoWeek").format("YYYY-MM-DD"))
const endDate = ref(dayjs().startOf("isoWeek").add(6, "day").format("YYYY-MM-DD"))
const mode = ref<"copy" | "move">("copy")
const entries = ref<OrganizeEntry[]>([])
const organizing = ref(false)
const showConfirm = ref(false)
const existingFiles = ref<string[]>([])

const columns = [
  { title: t("broadcast.date"), key: "date", width: 120 },
  { title: t("organize.slot"), key: "slotTime", width: 100 },
  { title: "原曲", key: "songTitle", width: 200 },
  {
    title: t("organize.targetName"),
    key: "targetName",
    render: (row: OrganizeEntry) => h(NInput, {
      value: row.targetName,
      onUpdateValue: (v: string) => { row.targetName = v },
    }),
  },
]

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
})

function dayIndexFromDate(dateStr: string): number {
  const d = dayjs(dateStr)
  return d.day() === 0 ? 7 : d.day()
}

function slotsForDate(dateStr: string): TimeSlot[] {
  const dayIdx = dayIndexFromDate(dateStr)
  return settingsStore.timeSlots.filter(s => s.dayIndex === dayIdx).sort((a, b) => a.order - b.order)
}

async function browse() {
  const path = await selectDir()
  if (path) targetDir.value = path
}

function generateEntries() {
  entries.value = []
  const start = dayjs(startDate.value)
  const end = dayjs(endDate.value)
  if (!start.isValid() || !end.isValid() || end.isBefore(start)) {
    message.error("日期范围错误")
    return
  }
  let cur = start
  let daySeq = 1
  while (cur.isBefore(end) || cur.isSame(end, "day")) {
    const date = cur.format("YYYY-MM-DD")
    const slots = slotsForDate(date)
    for (const slot of slots) {
      const song = songsStore.dormSongs.find(s => s.date === date && s.timeSlotId === slot.id)
      const targetName = `${String(daySeq).padStart(2, "0")}_${date}_${slot.time.replace(/:/g, "")}.mp3`
      entries.value.push({
        key: `${date}-${slot.id}`,
        date,
        slot,
        source: song?.filePath || "",
        targetName,
        songTitle: song?.title || "（静音占位）",
      })
    }
    daySeq++
    cur = cur.add(1, "day")
  }
}

async function doOrganize(confirm = false) {
  if (entries.value.length === 0) return
  if (!targetDir.value) { message.error("请选择目标文件夹"); return }
  organizing.value = true
  try {
    const result = await organizeFiles({
      entries: entries.value.map(e => ({ source: e.source, targetName: e.targetName })),
      targetDir: targetDir.value,
      mode: mode.value,
      confirm,
    })
    if (result.confirmNeeded) {
      existingFiles.value = result.existingFiles || []
      showConfirm.value = true
      return
    }
    const ok = result.successful?.length || 0
    const fail = result.failed?.length || 0
    message.success(t("organize.success", { count: ok }))
    if (fail > 0) message.error(t("organize.failed", { count: fail }))
  } catch (err: any) {
    message.error(err?.message || "整理失败")
  } finally {
    organizing.value = false
  }
}

</script>
