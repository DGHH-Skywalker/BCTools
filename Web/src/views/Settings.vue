<template>
  <n-layout has-sider class="settings-layout">
    <n-layout-sider bordered collapse-mode="width" :collapsed-width="52" :width="176" show-trigger="bar">
      <div class="settings-sider">
        <n-button quaternary block class="back-button" @click="returnToApp">
          <template #icon><Left theme="outline" :size="18" :strokeWidth="3" /></template>
          返回主界面
        </n-button>
        <n-menu :value="route.path" :options="menuOptions" @update:value="navigate" />
      </div>
    </n-layout-sider>
    <n-layout-content content-style="padding:16px;"><router-view /></n-layout-content>
  </n-layout>
</template>
<script setup lang="ts">
import { computed, h } from "vue"
import { useRoute, useRouter } from "vue-router"
import type { MenuOption } from "naive-ui"
import { Calendar, Export, GuideBoard, Setting, Broadcast, Left } from "@icon-park/vue-next"
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
function returnToApp() {
  if (window.opener && !window.opener.closed) {
    window.opener.focus()
    window.close()
    window.setTimeout(() => router.push("/home"), 100)
    return
  }
  router.push("/home")
}
</script>
<style scoped>
.settings-layout { min-height: 100%; height: 100%; }
.settings-sider { display: flex; flex-direction: column; height: 100%; padding-top: 8px; }
.back-button { margin: 0 8px 8px; width: calc(100% - 16px); }
</style>
