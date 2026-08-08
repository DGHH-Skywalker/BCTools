<template>
  <n-space wrap>
    <n-tooltip v-if="!canExportDorm" trigger="hover">
      <template #trigger>
        <n-button type="primary" dashed disabled>{{ t("export.exportDorm") }}</n-button>
      </template>
      {{ t("export.needBackground") }}
    </n-tooltip>
    <n-button v-else type="primary" dashed :loading="exporting" @click="$emit('export-dorm')">{{ t("export.exportDorm") }}</n-button>

    <n-tooltip v-if="!canExportBroadcast" trigger="hover">
      <template #trigger>
        <n-button type="primary" dashed disabled>{{ t("export.exportBroadcast") }}</n-button>
      </template>
      {{ t("export.needBackground") }}
    </n-tooltip>
    <n-button v-else type="primary" dashed :loading="exporting" @click="$emit('export-broadcast')">{{ t("export.exportBroadcast") }}</n-button>
  </n-space>
</template>

<script setup lang="ts">
import { useI18n } from "../../i18n"

const { t } = useI18n()

defineProps<{
  exporting: boolean
  canExportDorm: boolean
  canExportBroadcast: boolean
}>()

defineEmits<{
  (e: "export-dorm"): void
  (e: "export-broadcast"): void
}>()
</script>
