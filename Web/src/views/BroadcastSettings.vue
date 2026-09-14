<template>
  <n-space vertical size="large">
    <div><n-h2 style="margin:0">播音栏目</n-h2><n-text depth="3">设置每一天在播音歌单中显示的栏目名称。</n-text></div>
    <n-grid cols="1 s:2 m:3" :x-gap="16" :y-gap="16"><n-grid-item v-for="day in 7" :key="day"><n-form-item :label="weekdayLabel(day)"><n-input v-model:value="columnMap[String(day)]" /></n-form-item></n-grid-item></n-grid>
    <n-space justify="end"><n-button type="primary" :loading="saving" @click="save">保存</n-button></n-space>
  </n-space>
</template>
<script setup lang="ts">
import { onMounted, reactive, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
const { weekdayLabel } = useI18n(); const settingsStore = useSettingsStore(); const message = useMessage()
const saving = ref(false); const columnMap = reactive<Record<string, string>>({})
onMounted(async () => { await settingsStore.fetchSettings(); Object.assign(columnMap, settingsStore.broadcastColumnMap) })
async function save() { saving.value = true; try { await settingsStore.updateSettings({ broadcastColumnMap: { ...columnMap } } as any); message.success("已保存") } finally { saving.value = false } }
</script>
