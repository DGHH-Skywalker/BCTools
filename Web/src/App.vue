<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <router-view />
          <AudioPlayerWidget v-show="!isHome" />
          <AppHeartbeat />
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { onMounted, ref, computed, watch } from "vue"
import { useRoute } from "vue-router"
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from "naive-ui"
import { domToPng } from "modern-screenshot"
import { loadAppConfig, useAppConfig } from "./composables/useAppConfig"
import { useTheme } from "./composables/useTheme"
import AudioPlayerWidget from "./components/common/AudioPlayerWidget.vue"
import AppHeartbeat from "./components/common/AppHeartbeat.vue"

const { config } = useAppConfig()
const route = useRoute()
const isHome = computed(() => route.path === "/home")
const { themeOverrides, theme, isDark, applyTheme } = useTheme()

watch(
  isDark,
  (dark) => {
    document.documentElement.setAttribute("data-theme", dark ? "dark" : "light")
  },
  { immediate: true },
)

onMounted(async () => {
  document.title = "小播点歌工具"
  const cfg = await loadAppConfig()
  applyTheme(cfg.themeColor)
  try {
    const div = document.createElement("div")
    div.style.cssText = "width:1px;height:1px;opacity:0;position:fixed;top:-999px"
    document.body.appendChild(div)
    domToPng(div).then(() => {
      document.body.removeChild(div)
    }).catch(() => {
      document.body.removeChild(div)
    })
  } catch {}
})
</script>

<style>
@font-face {
  font-family: 'JiangxiZhuokai';
  src: url('/fonts/江西拙楷3.0.ttf') format('truetype');
  font-weight: normal;
  font-style: normal;
}

:root {
  --theme-color: #0086C3;
  --theme-color-light: #4db8e840;
  --font-title: 'JiangxiZhuokai', 'STKaiti', 'KaiTi', serif;
}
</style>
