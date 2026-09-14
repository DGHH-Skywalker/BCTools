<template>
  <n-space vertical size="large"><div><n-h2 style="margin:0">导出偏好</n-h2><n-text depth="3">在这里统一管理海报导出的模式、背景和显示效果。</n-text></div><n-button type="primary" @click="show = true">编辑导出偏好</n-button><ExportAdvancedDrawer v-model:show="show" :drawer-width="drawerWidth" /></n-space>
</template>
<script setup lang="ts">
import { ref } from "vue"
import ExportAdvancedDrawer from "../components/export/ExportAdvancedDrawer.vue"
const show = ref(false); const drawerWidth = window.innerWidth <= 768 ? "100%" : "50%"
</script>
