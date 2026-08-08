<template>
  <div class="calendar-wrapper" :class="[props.size]">
    <n-config-provider :locale="locale">
      <n-calendar
        :value="calendarValue"
        :default-value="undefined"
        :is-date-disabled="isDateDisabled"
        :theme-overrides="calendarThemeOverrides"
        @update:value="onValueUpdate"
        @panel-change="onPanelChange"
      >
        <template #default="{ year: y, month: m, date: d }">
          <div
            class="calendar-cell"
            :class="{
              selected: isCurrentMonth(y, m) && isSelected(y, m, d),
              'has-songs': isCurrentMonth(y, m) && cellInfo(y, m, d).filledSlots > 0,
              'other-month': !isCurrentMonth(y, m),
            }"
          >
            <span v-if="isCurrentMonth(y, m)" class="cell-date-number">{{ d }}</span>
            <div
              v-if="isCurrentMonth(y, m) && cellInfo(y, m, d).totalSlots > 0"
              class="slot-badge"
              :class="cellInfo(y, m, d).filledSlots === cellInfo(y, m, d).totalSlots ? 'full' : 'partial'"
            >
              {{ cellInfo(y, m, d).filledSlots }}/{{ cellInfo(y, m, d).totalSlots }}
            </div>
          </div>
        </template>
      </n-calendar>
    </n-config-provider>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import dayjs from "dayjs"
import { zhCN, dateZhCN } from "naive-ui"

const locale = {
  ...zhCN,
  dateLocale: {
    ...dateZhCN,
    firstDayOfWeek: 1,
  },
}

const props = withDefaults(defineProps<{
  year: number
  month: number
  songsMap: Record<string, { totalSlots: number; filledSlots: number }>
  selectionMode?: "single" | "multiple"
  selectedDates?: string[]
  size?: "default" | "small" | "mini"
}>(), {
  size: "default",
})

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

const calendarThemeOverrides = computed(() => {
  if (props.size === "mini") {
    return {
      fontSize: "12px",
      titleFontSize: "14px",
      lineHeight: "1.2",
    }
  }
  if (props.size === "small") {
    return {
      fontSize: "13px",
      titleFontSize: "15px",
      lineHeight: "1.25",
    }
  }
  return {
    fontSize: "14px",
    titleFontSize: "16px",
    lineHeight: "1.4",
  }
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

function isCurrentMonth(y: number, m: number): boolean {
  return y === props.year && m === props.month
}

function isDateDisabled(ts: number): boolean {
  const d = dayjs(ts)
  return d.year() !== props.year || d.month() + 1 !== props.month
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

function onPanelChange(info: { year: number; month: number }) {
  emit("update:year", info.year)
  emit("update:month", info.month)
}
</script>

<style scoped>
.calendar-wrapper {
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
}

/* Naive UI 的日历高度决定格子大小：.n-calendar 固定 720px，.n-calendar-dates 使用 grid-auto-rows:1fr */
.calendar-wrapper :deep(.n-calendar) {
  height: 520px;
}

.calendar-wrapper :deep(.n-calendar-cell) {
  border: 1px solid #e0e0e0;
  padding: 0;
}

.calendar-wrapper :deep(.n-calendar-date) {
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.calendar-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  min-height: 36px;
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

.calendar-cell.other-month {
  background: transparent;
  cursor: default;
}

.calendar-cell.other-month:hover {
  background: transparent;
}

.slot-badge {
  margin-top: 2px;
  font-size: 13px;
  padding: 2px 6px;
  border-radius: 10px;
  line-height: 1.4;
  font-weight: 500;
}

.slot-badge.partial {
  background: #fff7e6;
  color: #fa8c16;
}

.slot-badge.full {
  background: #f6ffed;
  color: #52c41a;
}

/* 日期数字：slot 自己渲染，隐藏 n-calendar 自带的日期数字避免重复 */
.cell-date-number {
  font-size: 14px;
  line-height: 1;
  margin-bottom: 2px;
  color: #333;
}

/* 隐藏 n-calendar 自带的日期数字，日期数字由 slot 内 .cell-date-number 渲染，避免重复。
   naive-ui 在 .n-calendar-date__date 上设置了 display:flex，且其样式注入晚于组件
   scoped 样式、选择器特异性相当，需 !important 才能可靠覆盖，否则同一天会显示两个日期号。 */
.calendar-wrapper :deep(.n-calendar-date__date) {
  display: none !important;
}

/* 星期标题：参考 BroadcastGrid.vue 的矩形卡片风格 */
.calendar-wrapper :deep(.n-calendar-date__day) {
  border-radius: 8px;
  background: var(--theme-color-bg, #f5f5f5);
  color: var(--theme-color, #66ccff);
  font-family: "JiangxiZhuokai", "Microsoft YaHei", "PingFang SC", sans-serif;
  padding: 4px 0;
  text-align: center;
}

/* 小尺寸日历 */
.calendar-wrapper.small :deep(.n-calendar) {
  height: 420px;
}

.calendar-wrapper.small :deep(.n-calendar-date) {
  min-height: 36px;
}

.calendar-wrapper.small .calendar-cell {
  min-height: 28px;
}

.calendar-wrapper.small .slot-badge {
  font-size: 12px;
  padding: 2px 5px;
}

.calendar-wrapper.small .cell-date-number {
  font-size: 13px;
}

/* 迷你尺寸日历 */
.calendar-wrapper.mini :deep(.n-calendar) {
  height: 340px;
}

.calendar-wrapper.mini :deep(.n-calendar-date) {
  min-height: 32px;
}

.calendar-wrapper.mini .calendar-cell {
  min-height: 24px;
}

.calendar-wrapper.mini .slot-badge {
  font-size: 11px;
  padding: 1px 4px;
}

.calendar-wrapper.mini .cell-date-number {
  font-size: 12px;
}

@media (max-width: 768px) {
  .calendar-wrapper {
    max-width: 100%;
  }
  .calendar-wrapper :deep(.n-calendar) {
    height: 400px;
  }
  .calendar-wrapper :deep(.n-calendar-date) {
    min-height: 36px;
  }
  .calendar-cell {
    min-height: 32px;
  }
  .calendar-wrapper.small :deep(.n-calendar),
  .calendar-wrapper.mini :deep(.n-calendar) {
    height: 300px;
  }
}
</style>
