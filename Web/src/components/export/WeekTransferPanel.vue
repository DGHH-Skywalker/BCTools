<template>
  <div class="week-transfer-panel">
    <n-empty v-if="visibleGroups.length === 0" size="small" description="无匹配日期" />
    <n-collapse v-else :expanded-names="expandedNames" @update:expanded-names="expandedNames = $event">
      <n-collapse-item
        v-for="group in visibleGroups"
        :key="group.weekKey"
        :name="group.weekKey"
        :id="group.weekKey === currentWeekKey ? 'week-current' : undefined"
      >
        <template #header>
          <div class="week-header" @click.stop>
            <span class="week-label">{{ group.label }}</span>
            <span class="week-count">({{ group.selectedCount }}/{{ group.options.length }})</span>
          </div>
        </template>
        <div class="day-list">
          <n-checkbox
            v-for="opt in group.options"
            :key="opt.value"
            :checked="isSelected(opt.value)"
            class="day-checkbox"
            @update:checked="toggleDay(opt.value, $event)"
          >
            {{ opt.label }}
          </n-checkbox>
        </div>
      </n-collapse-item>
    </n-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { dayjs } from "../../utils/datetime"
import { NCollapse, NCollapseItem, NEmpty } from "naive-ui"


interface DayOption {
  label: string
  value: string
}

const props = defineProps<{
  options: DayOption[]
  selectedOptions: DayOption[]
  pattern: string
}>()

const emit = defineEmits<{
  update: [values: string[]]
}>()

const selectedSet = computed(() => new Set(props.selectedOptions.map((o) => o.value)))
const currentWeekKey = computed(() => {
  const now = dayjs()
  return `${now.isoWeekYear()}-W${now.isoWeek()}`
})

const groups = computed(() => {
  const map = new Map<string, { weekKey: string; label: string; options: DayOption[]; selectedCount: number }>()
  for (const opt of props.options) {
    const d = dayjs(opt.value)
    const weekKey = `${d.isoWeekYear()}-W${d.isoWeek()}`
    let group = map.get(weekKey)
    if (!group) {
      const week = d.isoWeek()
      const start = d.startOf("isoWeek")
      const end = d.endOf("isoWeek")
      group = {
        weekKey,
        label: `第${week}周 ${start.format("MM.DD")}-${end.format("MM.DD")}`,
        options: [],
        selectedCount: 0,
      }
      map.set(weekKey, group)
    }
    group.options.push(opt)
    if (selectedSet.value.has(opt.value)) group.selectedCount++
  }
  return Array.from(map.values())
})

const visibleGroups = computed(() => {
  const p = props.pattern.trim().toLowerCase()
  if (!p) return groups.value
  return groups.value.filter((g) => {
    if (g.label.toLowerCase().includes(p)) return true
    return g.options.some((o) => o.label.toLowerCase().includes(p))
  })
})

const expandedNames = ref<string[]>([])

watch(
  () => props.selectedOptions,
  () => {
    const keys = new Set(expandedNames.value)
    for (const g of groups.value) {
      if (g.selectedCount > 0 && g.selectedCount < g.options.length) {
        keys.add(g.weekKey)
      }
    }
    expandedNames.value = Array.from(keys)
  },
  { immediate: true, deep: true }
)

function isSelected(value: string): boolean {
  return selectedSet.value.has(value)
}

function toggleDay(value: string, checked: boolean) {
  const next = new Set(selectedSet.value)
  if (checked) next.add(value)
  else next.delete(value)
  emit("update", Array.from(next))
}

function toggleWeek(group: { options: DayOption[] }, checked: boolean) {
  const next = new Set(selectedSet.value)
  for (const o of group.options) {
    if (checked) next.add(o.value)
    else next.delete(o.value)
  }
  emit("update", Array.from(next))
}
</script>

<style scoped>
.week-transfer-panel {
  padding: 8px 12px;
}

.week-header {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: default;
}

.week-label {
  font-weight: 500;
}

.week-count {
  color: #888;
  font-size: 12px;
}

.day-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
  padding: 4px 0 8px 24px;
}

.day-checkbox {
  line-height: 1.4;
}
</style>
