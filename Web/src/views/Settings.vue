<template>
  <n-layout has-sider class="settings-layout">
    <n-layout-sider bordered collapse-mode="width" :collapsed-width="52" :width="176" show-trigger="bar">
      <n-menu :value="route.path" :options="menuOptions" @update:value="navigate" />
    </n-layout-sider>
    <n-layout-content content-style="padding:16px;"><router-view /></n-layout-content>
  </n-layout>
</template>
<script setup lang="ts">
import { computed, h } from "vue"
import { useRoute, useRouter } from "vue-router"
import type { MenuOption } from "naive-ui"
import { Calendar, Export, GuideBoard, Setting, Broadcast } from "@icon-park/vue-next"
const route = useRoute(); const router = useRouter()
const iconProps = { theme: "outline" as const, size: 18, strokeWidth: 3 }
const menuOptions = computed<MenuOption[]>(() => [
  { key: "/settings", label: "软件信息", icon: () => h(Setting, iconProps) },
  { key: "/settings/slots", label: "宿舍时段与查重", icon: () => h(Calendar, iconProps) },
  { key: "/settings/broadcast", label: "播音栏目", icon: () => h(Broadcast, iconProps) },
  { key: "/settings/export", label: "导出偏好", icon: () => h(Export, iconProps) },
  { key: "/settings/guide", label: "软件指南", icon: () => h(GuideBoard, iconProps) },
])
function navigate(path: string) { router.push(path) }
</script>
<style scoped>.settings-layout { min-height: 100%; height: 100%; }</style>
