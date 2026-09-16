<template>
  <n-space vertical size="large">
    <div><n-h2 style="margin:0">歌单背景设置</n-h2><n-text depth="3">在这里统一管理歌单海报的背景、版式和显示效果。</n-text></div>
    <ExportPreferencesPanel />
  </n-space>
</template>
<script setup lang="ts">
import ExportPreferencesPanel from "../components/export/ExportPreferencesPanel.vue"
</script>
