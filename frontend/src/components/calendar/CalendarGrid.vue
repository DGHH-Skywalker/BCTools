<template>
  <div class="calendar-grid" @keydown="handleKeydown" tabindex="0">
    <div class="calendar-header">
      <n-button @click="prevMonth">&lt;</n-button>
      <n-h4 style="margin:0;">{{ year }}年{{ month }}月</n-h4>
      <n-button @click="nextMonth">&gt;</n-button>
      <n-button size="small" @click="goToday">{{ t("common.today") }}</n-button>
    </div>
    <div class="weekday-row">
      <div v-for="(day, i) in weekdays" :key="i" class="weekday-cell">{{ day }}</div>
    </div>
    <div class="days-grid">
      <div v-for="(cell, i) in cells" :key="i"
        :class="['day-cell', { selected: isSelected(cell.date), 'other-month': !cell.isCurrent, empty: cell.slots === 0, 'has-songs': cell.filled > 0 }]"
        @click="onDateClick(cell.date)">
        <span class="day-num">{{ cell.day }}</span>
        <n-tag v-if="cell.slots > 0" :type="cell.filled === cell.slots ? 'success' : 'warning'" size="tiny">
          {{ cell.filled }}/{{ cell.slots }}
        </n-tag>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from "vue"
import { useI18n } from "../../i18n"
import { useSettingsStore } from "../../stores/settings"

const props = defineProps<{
  year: number
  month: number
  songsMap: Record<string, { totalSlots: number; filledSlots: number }>
  selectionMode?: "single" | "multiple"
  selectedDates?: string[]
}>()

const emit = defineEmits<{
  dateClick: [date: string]
  selectionChange: [dates: string[]]
  escape: []
  "update:year": [year: number]
  "update:month": [month: number]
}>()

const { t, weekdayShortName, dayIndexFromDate } = useI18n()
const settingsStore = useSettingsStore()

const weekdays = computed(() => {
  // Return 日,一,二,三,四,五,六 (Sunday first)
  const names = ["日","一","二","三","四","五","六"]
  return names.map((_, i) => weekdayShortName(i === 0 ? 7 : i) || names[i])
})

const cells = computed(() => {
  const firstDay = new Date(props.year, props.month - 1, 1)
  const lastDay = new Date(props.year, props.month, 0)
  const startWeekday = firstDay.getDay()
  const daysInMonth = lastDay.getDate()
  const result: any[] = []
  for (let i = 0; i < startWeekday; i++) {
    const d = new Date(props.year, props.month - 1, -startWeekday + i + 1)
    result.push({ day: d.getDate(), date: formatDate(d), isCurrent: false, slots: 0, filled: 0 })
  }
  for (let d = 1; d <= daysInMonth; d++) {
    const date = formatDate(new Date(props.year, props.month - 1, d))
    const info = props.songsMap[date]
    result.push({ day: d, date, isCurrent: true, slots: info?.totalSlots ?? 0, filled: info?.filledSlots ?? 0 })
  }
  // Remove last row if all cells are from next month
  while (result.length > 35 && result.length % 7 === 0) {
    const lastRow = result.slice(-7)
    if (lastRow.every((c: any) => !c.isCurrent)) { result.splice(-7) } else { break }
  }
  return result
})

function formatDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`
}

function onDateClick(date: string) {
  emit("dateClick", date)
}

function isSelected(date: string): boolean {
  return props.selectedDates?.includes(date) ?? false
}

function prevMonth() {
  if (props.month <= 1) {
    emit("update:year", props.year - 1)
    emit("update:month", 12)
  } else {
    emit("update:month", props.month - 1)
  }
}

function nextMonth() {
  if (props.month >= 12) {
    emit("update:year", props.year + 1)
    emit("update:month", 1)
  } else {
    emit("update:month", props.month + 1)
  }
}

function goToday() {
  emit("dateClick", formatDate(new Date()))
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === "Escape") emit("escape")
}

defineExpose({ goToday })
</script>

<style scoped>
.calendar-grid { outline: none; }
.calendar-header { display:flex;align-items:center;gap:8px;padding:8px; }
.weekday-row { display:grid;grid-template-columns:repeat(7,1fr);text-align:center;font-weight:bold;padding:4px 0; }
.days-grid { display:grid;grid-template-columns:repeat(7,1fr);gap:2px; }
.day-cell { display:flex;flex-direction:column;align-items:center;padding:8px;border-radius:4px;cursor:pointer;min-height:60px; }
.day-cell:hover { background:#f0f0f0; }
.day-cell.selected { border:2px solid var(--theme-color, #66ccff); }
.day-cell.has-songs { background:var(--theme-color-light, #66ccff40); }
.day-cell.other-month { opacity:0.3; }
.day-num { font-size:14px; }
@media (max-width:768px) { .day-cell { min-height:40px;padding:4px;font-size:12px; } }
</style>
