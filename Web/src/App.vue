<template>
  <n-config-provider :theme-overrides="themeOverrides">
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
import { onMounted, ref, computed } from "vue"
import { useRoute } from "vue-router"
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from "naive-ui"
import { domToPng } from "modern-screenshot"
import { loadAppConfig, useAppConfig } from "./composables/useAppConfig"
import AudioPlayerWidget from "./components/common/AudioPlayerWidget.vue"
import AppHeartbeat from "./components/common/AppHeartbeat.vue"

const { config } = useAppConfig()
const route = useRoute()
const isHome = computed(() => route.path === "/home")
const THEME_COLOR = "#0086C3"
const themeOverrides = ref({
  common: {
    primaryColor: THEME_COLOR,
    primaryColorHover: "#339fd2",
    primaryColorPressed: "#006fa2",
    primaryColorSuppl: THEME_COLOR,
  },
})

function hexToRgb(hex: string) {
  const v = hex.replace("#", "")
  const full = v.length === 3 ? v.split("").map(c => c + c).join("") : v
  const num = parseInt(full, 16)
  return { r: (num >> 16) & 255, g: (num >> 8) & 255, b: num & 255 }
}

function rgbToHex(r: number, g: number, b: number) {
  return "#" + [r, g, b].map(x => Math.max(0, Math.min(255, x)).toString(16).padStart(2, "0")).join("")
}

function adjustColor(hex: string, amount: number) {
  const { r, g, b } = hexToRgb(hex)
  return rgbToHex(r + amount, g + amount, b + amount)
}

function applyTheme(color: string) {
  themeOverrides.value = {
    common: {
      primaryColor: color,
      primaryColorHover: adjustColor(color, 25),
      primaryColorPressed: adjustColor(color, -25),
      primaryColorSuppl: color,
    },
  }
  document.documentElement.style.setProperty("--theme-color", color)
  document.documentElement.style.setProperty("--theme-color-light", adjustColor(color, 60) + "40")
}

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

@media (prefers-color-scheme: dark) {
  .n-button--dashed-type-primary {
    color: #fff !important;
    border-color: #fff !important;
  }
  .n-button--dashed-type-primary:not(.n-button--disabled):hover {
    color: rgba(255, 255, 255, 0.85) !important;
    border-color: rgba(255, 255, 255, 0.85) !important;
  }
  .n-button--dashed-type-primary:not(.n-button--disabled):focus {
    color: rgba(255, 255, 255, 0.85) !important;
    border-color: rgba(255, 255, 255, 0.85) !important;
  }
}
</style>
