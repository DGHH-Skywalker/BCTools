<template>
  <section class="broadcast-settings">
    <header class="page-header">
      <div>
        <n-h2 style="margin:0">播音栏目</n-h2>
        <n-text depth="3">设置每一天在播音歌单中显示的栏目名称。</n-text>
      </div>
      <n-button type="primary" :loading="saving" @click="save">保存</n-button>
    </header>

    <n-form label-placement="left" label-width="52" label-align="left">
      <div class="column-grid">
        <n-form-item v-for="day in 7" :key="day" :label="weekdayLabel(day)" :show-feedback="false">
          <n-input v-model:value="columnMap[String(day)]" size="small" />
        </n-form-item>
      </div>
    </n-form>
  </section>
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

<style scoped>
.broadcast-settings { display: flex; flex-direction: column; gap: 14px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.column-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 10px 18px; max-width: 1040px; }
.column-grid :deep(.n-form-item) { min-width: 0; }
@media (max-width: 560px) {
  .page-header { align-items: flex-end; }
  .column-grid { grid-template-columns: 1fr; }
}
</style>
