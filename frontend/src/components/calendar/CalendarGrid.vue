<template>
  <div class="calendar-wrapper">
    <n-calendar
      :value="calendarValue"
      :default-value="undefined"
      :is-date-disabled="isDateDisabled"
      @update:value="onValueUpdate"
      @panel-change="onPanelChange"
    >
      <template #default="{ year: y, month: m, date: d }">
        <div class="calendar-cell" :class="{ selected: isSelected(y, m, d), 'has-songs': cellInfo(y, m, d).filledSlots > 0 }">
          <div class="day-num">{{ d }}</div>
          <div v-if="cellInfo(y, m, d).totalSlots > 0" class="slot-badge" :class="cellInfo(y, m, d).filledSlots === cellInfo(y, m, d).totalSlots ? 'full' : 'partial'">
            {{ cellInfo(y, m, d).filledSlots }}/{{ cellInfo(y, m, d).totalSlots }}
          </div>
        </div>
      </template>
    </n-calendar>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import dayjs from "dayjs"

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
  "update:year": [year: number]
  "update:month": [month: number]
}>()

const selectedSet = ref(new Set<string>(props.selectedDates || []))
const panelTs = ref(dayjs(`${props.year}-${String(props.month).padStart(2, "0")}-01`).valueOf())

watch(() => props.selectedDates, (dates) => {
  selectedSet.value = new Set(dates || [])
}, { immediate: true, deep: true })

watch(() => [props.year, props.month], ([y, m]) => {
  panelTs.value = dayjs(`${y}-${String(m as number).padStart(2, "0")}-01`).valueOf()
}, { immediate: true })

const calendarValue = computed(() => {
  if (props.selectionMode === "multiple") return undefined
  if (selectedSet.value.size === 1) {
    const d = Array.from(selectedSet.value)[0]
    return dayjs(d).valueOf()
  }
  return undefined
})

function formatDateKey(y: number, m: number, d: number): string {
  return `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`
}

function cellInfo(y: number, m: number, d: number) {
  return props.songsMap[formatDateKey(y, m, d)] || { totalSlots: 0, filledSlots: 0 }
}

function isSelected(y: number, m: number, d: number): boolean {
  return selectedSet.value.has(formatDateKey(y, m, d))
}

function isDateDisabled(ts: number): boolean {
  return false
}

function onValueUpdate(ts: number) {
  const date = dayjs(ts).format("YYYY-MM-DD")
  if (props.selectionMode === "multiple") {
    const key = date
    const next = new Set(selectedSet.value)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    selectedSet.value = next
    emit("selectionChange", Array.from(next))
  } else {
    selectedSet.value = new Set([date])
    emit("dateClick", date)
  }
}

function onPanelChange(info: { year: number; month: number; date: number }) {
  emit("update:year", info.year)
  emit("update:month", info.month)
}
</script>

<style scoped>
.calendar-wrapper {
  max-width: 520px;
  margin: 0 auto;
}

.calendar-wrapper :deep(.n-calendar-cell) {
  border: 1px solid #e0e0e0;
  padding: 0;
}

.calendar-wrapper :deep(.n-calendar-date) {
  min-height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.calendar-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 48px;
  width: 100%;
  padding: 2px;
  cursor: pointer;
  border-radius: 4px;
  transition: background 0.15s;
}

.calendar-cell:hover {
  background: #f5f5f5;
}

.calendar-cell.selected {
  background: var(--theme-color-light, #66ccff40);
  outline: 1px solid var(--theme-color, #66ccff);
}

.calendar-cell.has-songs:not(.selected) {
  background: #f0f9ff;
}

.day-num {
  font-size: 13px;
  line-height: 1.2;
}

.slot-badge {
  margin-top: 1px;
  font-size: 10px;
  padding: 0 3px;
  border-radius: 8px;
  line-height: 1.3;
}

.slot-badge.partial {
  background: #fff7e6;
  color: #fa8c16;
}

.slot-badge.full {
  background: #f6ffed;
  color: #52c41a;
}

@media (max-width: 768px) {
  .calendar-wrapper {
    max-width: 100%;
  }
  .calendar-wrapper :deep(.n-calendar-date) {
    min-height: 48px;
  }
  .calendar-cell {
    min-height: 42px;
  }
}
</style>
