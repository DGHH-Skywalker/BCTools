<template>
  <n-config-provider :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <router-view />
          <AudioPlayerWidget />
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from "naive-ui"
import { domToPng } from "modern-screenshot"
import { loadAppConfig, useAppConfig } from "./composables/useAppConfig"
import AudioPlayerWidget from "./components/common/AudioPlayerWidget.vue"

const { config } = useAppConfig()
const themeOverrides = ref({
  common: {
    primaryColor: "#66ccff",
    primaryColorHover: "#85dbff",
    primaryColorPressed: "#4db8e8",
    primaryColorSuppl: "#66ccff",
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
  // Auto-save and shutdown when page is closed
  window.addEventListener("beforeunload", () => {
    try {
      navigator.sendBeacon("/api/shutdown", "")
    } catch(e) {
      // Best-effort shutdown notification
    }
  })
  document.title = "广播站歌单工具"
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
:root {
  --theme-color: #66ccff;
  --theme-color-light: #66ccff40;
}
</style>
