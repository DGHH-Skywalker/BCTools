<template>
  <n-layout has-sider class="settings-layout">
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="52"
      :width="176"
      show-trigger="bar"
      :collapsed="collapsed"
      @collapse="collapsed = true"
      @expand="collapsed = false"
    >
      <div class="settings-sider">
        <n-button quaternary block class="back-button" title="返回主界面" aria-label="返回主界面" @click="returnToApp">
          <template #icon><Left theme="outline" :size="18" :strokeWidth="3" /></template>
          <span v-if="!collapsed">返回主界面</span>
        </n-button>
        <n-menu
          :collapsed="collapsed"
          :collapsed-width="52"
          :collapsed-icon-size="20"
          :value="route.path"
          :options="menuOptions"
          @update:value="navigate"
        />
      </div>
    </n-layout-sider>
    <n-layout-content content-style="padding:16px;"><router-view /></n-layout-content>
  </n-layout>
</template>
<script setup lang="ts">
import { computed, h, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import type { MenuOption } from "naive-ui"
import { Calendar, Export, GuideBoard, Setting, Broadcast, Left } from "@icon-park/vue-next"
const route = useRoute(); const router = useRouter()
const collapsed = ref(false)
const iconProps = { theme: "outline" as const, size: 18, strokeWidth: 3 }
const menuOptions = computed<MenuOption[]>(() => [
  { key: "/settings", label: "软件信息", icon: () => h(Setting, iconProps) },
  { key: "/settings/slots", label: "时间配置", icon: () => h(Calendar, iconProps) },
  { key: "/settings/broadcast", label: "播音栏目", icon: () => h(Broadcast, iconProps) },
  { key: "/settings/export", label: "歌单背景设置", icon: () => h(Export, iconProps) },
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
