<template>
  <n-space vertical size="large">
    <div><n-h2 style="margin:0">导出偏好</n-h2><n-text depth="3">在这里统一管理海报导出的模式、背景和显示效果。</n-text></div>
    <ExportPreferencesPanel />
  </n-space>
</template>
<script setup lang="ts">
import ExportPreferencesPanel from "../components/export/ExportPreferencesPanel.vue"
</script>
