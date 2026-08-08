<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="t('settings.broadcastColumnMap')"
    :style="{ width: '90%', maxWidth: '720px' }"
    :content-style="{ padding: '16px' }"
    @update:show="emit('update:show', $event)"
  >
    <n-grid cols="1 s:2 m:3 l:3 xl:3" :x-gap="16" :y-gap="16">
      <n-grid-item v-for="idx in 7" :key="idx">
        <n-space vertical style="width:100%;">
          <n-text strong>{{ weekdayLabel(idx) }}</n-text>
          <n-select v-model:value="selectValues[idx]" :options="columnOptions" @update:value="(v: string) => onColumnChange(idx, v)" />
          <n-input
            v-if="selectValues[idx] === 'custom'"
            v-model:value="customValues[idx]"
            :placeholder="t('settings.custom')"
            @update:value="(v: string) => onCustomChange(idx, v)"
          />
        </n-space>
      </n-grid-item>
    </n-grid>

    <template #footer>
      <n-space justify="end">
        <n-button @click="close">{{ t("common.cancel") }}</n-button>
        <n-button type="primary" :loading="saving" @click="save">{{ t("common.save") }}</n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch } from "vue"
import { useI18n } from "../../i18n"
import { useSettingsStore } from "../../stores/settings"

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()

const { t, weekdayLabel } = useI18n()
const settingsStore = useSettingsStore()

const presetColumns = ["新闻", "文学", "音乐", "电影"]
const columnOptions = [
  { label: "新闻", value: "新闻" },
  { label: "文学", value: "文学" },
  { label: "音乐", value: "音乐" },
  { label: "电影", value: "电影" },
  { label: "无", value: "" },
  { label: t("settings.custom"), value: "custom" },
]

const selectValues = ref<Record<number, string>>({})
const customValues = ref<Record<number, string>>({})
const saving = ref(false)

function refreshFromStore() {
  for (let i = 1; i <= 7; i++) {
    const val = settingsStore.broadcastColumnMap[String(i)] || ""
    if (presetColumns.includes(val)) {
      selectValues.value[i] = val
      customValues.value[i] = ""
    } else if (val === "") {
      selectValues.value[i] = ""
      customValues.value[i] = ""
    } else {
      selectValues.value[i] = "custom"
      customValues.value[i] = val
    }
  }
}

watch(() => props.show, (show) => {
  if (show) refreshFromStore()
}, { immediate: true })

function onColumnChange(idx: number, value: string) {
  if (value === "custom") {
    customValues.value[idx] = customValues.value[idx] || ""
  } else {
    customValues.value[idx] = ""
  }
}

function onCustomChange(idx: number, value: string) {
  customValues.value[idx] = value
}

function close() {
  emit("update:show", false)
}

async function save() {
  saving.value = true
  try {
    const map: Record<string, string> = {}
    for (let i = 1; i <= 7; i++) {
      const key = String(i)
      if (selectValues.value[i] === "custom") {
        map[key] = customValues.value[i] || ""
      } else {
        map[key] = selectValues.value[i] || ""
      }
    }
    await settingsStore.updateSettings({ broadcastColumnMap: map } as any)
    close()
  } catch (err: any) {
    console.error(err)
  } finally {
    saving.value = false
  }
}
</script>
