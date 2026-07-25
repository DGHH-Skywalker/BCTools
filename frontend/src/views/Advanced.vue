<template>
  <div style="padding:24px;background:#fff;min-height:100vh;">
    <n-h2>{{ t("advanced.title") }}</n-h2>
    <n-collapse>
      <n-collapse-item v-for="day in 7" :key="day" :title="t('weekdays')[day-1]" :name="String(day)">
        <n-data-table :columns="slotColumns" :data="daySlots(day)" :bordered="true" :max-height="300" />
        <n-button dashed style="margin-top:8px;" @click="addSlot(day)">{{ t("advanced.addSlot") }}</n-button>
      </n-collapse-item>
    </n-collapse>
    <n-button style="margin-top:16px;" @click="exitAdvanced">{{ t("advanced.exit") }}</n-button>
  </div>
</template>

<script setup lang="ts">
import { h, onMounted, onUnmounted, ref } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import type { TimeSlot } from "../api/types"
import { NButton, NInputNumber, NTimePicker } from "naive-ui"
import { useMessage } from "naive-ui"

const { t, weekdayName } = useI18n()
const settingsStore = useSettingsStore()
const router = useRouter()
const message = useMessage()
let idleTimer: ReturnType<typeof setTimeout> | null = null

onMounted(() => {
  settingsStore.fetchSettings()
  resetIdleTimer()
  document.addEventListener("mousemove", resetIdleTimer)
  document.addEventListener("keydown", resetIdleTimer)
})

onUnmounted(() => {
  if (idleTimer) clearTimeout(idleTimer)
  document.removeEventListener("mousemove", resetIdleTimer)
  document.removeEventListener("keydown", resetIdleTimer)
})

function resetIdleTimer() {
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    sessionStorage.removeItem("advanced_authenticated")
    message.warning(t("advanced.locked"))
    router.push("/settings")
  }, 15 * 60 * 1000)
}

function daySlots(dayIndex: number): TimeSlot[] {
  return settingsStore.timeSlots.filter((s) => s.dayIndex === dayIndex).sort((a, b) => a.order - b.order)
}

const slotColumns = [
  { title: t("advanced.order"), key: "order", width: 80, render: (row: TimeSlot) => h(NInputNumber, { value: row.order, min: 1, style: "width:60px" }) },
  { title: t("advanced.time"), key: "time", width: 120, render: (row: TimeSlot) => h(NTimePicker, { value: parseTime(row.time), format: "HH:mm", style: "width:100px" }) },
  { title: "", key: "actions", width: 80, render: (row: TimeSlot) => h(NButton, { size: "small", type: "error", onClick: () => deleteSlot(row.id) }, () => t("advanced.deleteSlot")) },
]

function parseTime(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return new Date(1970, 0, 1, h, m).getTime()
}

function addSlot(dayIndex: number) {
  const slots = [...settingsStore.timeSlots]
  const tempId = crypto.randomUUID?.() || `temp-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  slots.push({ id: tempId, dayIndex, order: slots.filter((s) => s.dayIndex === dayIndex).length + 1, time: "12:00" })
  settingsStore.updateSettings({ timeSlots: slots } as any)
}

async function deleteSlot(id: string) {
  const slots = settingsStore.timeSlots.filter((s) => s.id !== id)
  const result = await settingsStore.updateSettings({ timeSlots: slots, confirmed: true } as any)
  if (result?.confirmNeeded) {
    message.info(t("advanced.deletedSongs", { count: result.affectedCount }))
  }
}

function exitAdvanced() {
  sessionStorage.removeItem("advanced_authenticated")
  router.push("/settings")
}
</script>
