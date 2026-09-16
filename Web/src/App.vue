<template>
  <n-config-provider :theme="theme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <router-view />
          <UpdateLogNotice />
          <AudioPlayerWidget v-show="!isHome" />
          <AppHeartbeat />
        </n-notification-provider>
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { onMounted, computed, watch } from "vue"
import { useRoute } from "vue-router"
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from "naive-ui"
import { domToPng } from "modern-screenshot"
import { loadAppConfig, useAppConfig } from "./composables/useAppConfig"
import { useTheme } from "./composables/useTheme"
import AudioPlayerWidget from "./components/common/AudioPlayerWidget.vue"
import AppHeartbeat from "./components/common/AppHeartbeat.vue"
import UpdateLogNotice from "./components/common/UpdateLogNotice.vue"

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

<!--
  全局样式：之前是 Web/src/styles/{variables,global}.css 拆出去的；
  现把 CSS 变量、基础 reset、字体 @font-face 全收回来，删掉 styles/ 目录，
  业务侧继续走 Naive UI 自带样式，Naive UI 无法覆盖的（海报导出 DOM / 字体）
  留在 App.vue 与对应 composable。
-->
<style>
@font-face {
  font-family: 'JiangxiZhuokai';
  src: url('/fonts/江西拙楷3.0.ttf') format('truetype');
  font-weight: normal;
  font-style: normal;
}

:root {
  /* Primary palette —— 同时被 Naive UI themeOverrides 引用 */
  --color-primary: #0086C3;
  --color-primary-hover: #339fd2;
  --color-primary-pressed: #006fa2;
  --color-primary-light: rgba(0, 134, 195, 0.1);
  --color-primary-light-strong: #4db8e840;

  /* Semantic colors */
  --color-warning: #d4a017;
  --color-warning-light: rgba(212, 160, 23, 0.12);
  --color-error: #e74c3c;
  --color-error-light: rgba(231, 76, 60, 0.12);
  --color-success: #27ae60;

  /* Neutral colors */
  --color-white: #ffffff;
  --color-black: #000000;
  --color-text: #1a1a1a;
  --color-text-secondary: #666666;
  --color-text-muted: #999999;
  --color-bg: #ffffff;
  --color-bg-light: #f5f7fa;
  --color-border: #e0e0e0;
  --color-border-dark: #dddddd;
  --color-overlay: rgba(0, 0, 0, 0.45);

  /* Export / poster specific */
  --color-poster-bg: #1a1a1a;
  --color-poster-text: #ffffff;
  --color-poster-muted: #d7d7d7;
  --color-poster-table-border: rgba(255, 255, 255, 0.85);
  --color-poster-table-border-broadcast: #ffffff;

  /* Spacing scale */
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
  --spacing-2xl: 48px;
  --spacing-3xl: 64px;

  /* Layout */
  --page-max-width: 1000px;
  --page-max-width-lg: 1200px;
  --page-max-width-xl: 1600px;
  --page-padding: var(--spacing-md);

  /* Typography */
  --font-title: 'JiangxiZhuokai', 'STKaiti', 'KaiTi', serif;
  --font-song: 'FZYanSong', '方正颜宋简体', 'STSong', 'SimSun', 'Songti SC', serif;
  --font-slot: 'FZXiaoBiaoSong', '方正小标宋简', 'STSong', 'SimSun', 'Songti SC', serif;
  --font-body: 'HarmonyOS Sans SC', 'HarmonyOS Sans', 'DengXian', 'Microsoft YaHei', 'PingFang SC', sans-serif;
  --font-fangsong: 'FangSong_GB2312', '仿宋_GB2312', 'FangSong', 'STFangsong', serif;

  /* Radius */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 24px;
  --radius-2xl: 40px;

  /* Shadows */
  --shadow-sm: 0 2px 6px rgba(0, 0, 0, 0.08);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.12);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.16);

  /* App specific */
  --theme-color: var(--color-primary);
  --theme-color-light: rgba(0, 134, 195, 0.25);
  --sider-width: 200px;
  --sider-collapsed-width: 64px;
  --header-height: 49px;
}

[data-theme="dark"] {
  --color-text: #f0f0f0;
  --color-text-secondary: #bbbbbb;
  --color-text-muted: #888888;
  --color-bg: #121212;
  --color-bg-light: #1e1e1e;
  --color-border: #333333;
  --color-border-dark: #444444;
  --color-overlay: rgba(0, 0, 0, 0.65);
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    --color-text: #f0f0f0;
    --color-text-secondary: #bbbbbb;
    --color-text-muted: #888888;
    --color-bg: #121212;
    --color-bg-light: #1e1e1e;
    --color-border: #333333;
    --color-border-dark: #444444;
    --color-overlay: rgba(0, 0, 0, 0.65);
  }
}

/* Base reset —— Naive UI 内部已经做了大部分归零，这里只补 box-sizing 与挂载根尺寸 */
*,
*::before,
*::after {
  box-sizing: border-box;
}

html,
body,
#app {
  margin: 0;
  padding: 0;
  width: 100%;
  height: 100%;
}

body {
  font-family: var(--font-body);
  color: var(--color-text);
  background-color: var(--color-bg);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Page container helpers —— 之前在 styles/global.css，现收回到使用方 PageContainer.vue */
.page-container {
  padding: var(--page-padding);
  max-width: var(--page-max-width);
  margin: 0 auto;
}

.page-container-lg {
  max-width: var(--page-max-width-lg);
}

.page-container-xl {
  max-width: var(--page-max-width-xl);
}
</style>
