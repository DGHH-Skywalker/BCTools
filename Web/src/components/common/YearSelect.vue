<template>
  <n-select
    :value="value"
    :options="yearOptions"
    virtual-scroll
    :style="{ width }"
    @update:value="(v) => $emit('update:value', v as number)"
  />
</template>

<script setup lang="ts">
import { computed } from "vue"

withDefaults(defineProps<{
  value: number
  width?: string
}>(), {
  width: "120px",
})

defineEmits<{
  "update:value": [value: number]
}>()

const yearOptions = computed(() => {
  const years: { label: string; value: number }[] = []
  for (let y = 2026; y <= 2100; y++) {
    years.push({ label: `${y}`, value: y })
  }
  return years
})

// Expose the same range for parent components that need it
const MIN_YEAR = 2026
const MAX_YEAR = 2100
</script>
