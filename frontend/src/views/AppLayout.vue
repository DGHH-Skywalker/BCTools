<template>
  <n-layout style="height:100vh;" has-sider>
    <n-layout-sider bordered :collapsed-width="64" :width="200" show-trigger="bar" collapse-mode="width" :collapsed="collapsed" @collapse="collapsed=true" @expand="collapsed=false">
      <n-h3 v-if="!collapsed" style="padding:12px 16px;margin:0;white-space:nowrap;font-style:normal;">{{ t("app.shortTitle") }}</n-h3>
      <n-menu :collapsed="collapsed" :collapsed-width="64" :collapsed-icon-size="22" :value="activeKey" :options="menuOptions" @update:value="onMenuChange" />
    </n-layout-sider>
    <n-layout style="height:100vh;">
      <n-layout-header v-if="config.languageSwitchEnabled" style="padding:8px 16px;background:#fff;border-bottom:1px solid #e0e0e0;display:flex;justify-content:flex-end;align-items:center;height:49px;">
        <n-button-group size="small">
          <n-button :type="currentLocale==='zh-CN'?'primary':'default'" @click="switchLang('zh-CN')">中</n-button>
          <n-button :type="currentLocale==='en'?'primary':'default'" @click="switchLang('en')">EN</n-button>
        </n-button-group>
      </n-layout-header>
      <n-layout-content
        :native-scrollbar="true"
        :scrollbar-props="{ trigger: 'none' }"
        :style="{ padding: 0, height: config.languageSwitchEnabled ? 'calc(100vh - 49px)' : '100vh' }"
      >
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>
<script setup lang="ts">
import { ref, computed, h } from "vue"
import { useRouter, useRoute } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useAppConfig } from "../composables/useAppConfig"
import type { MenuOption } from "naive-ui"
import { NIcon } from "naive-ui"

const router = useRouter(); const route = useRoute()
const { t, setLocale, currentLocale } = useI18n()
const settingsStore = useSettingsStore()
const { config } = useAppConfig()
const collapsed = ref(false)

const menuOptions: MenuOption[] = [
  { key: "/home", label: () => t("nav.home"), icon: () => h(NIcon, null, "🏠") },
  { key: "/song/import", label: () => t("nav.songImport"), icon: () => h(NIcon, null, "📥") },
  { key: "/dorm/manage", label: () => t("nav.dormManage"), icon: () => h(NIcon, null, "📋") },
  { key: "/broadcast", label: () => t("nav.broadcast"), icon: () => h(NIcon, null, "📢") },
  { key: "/export", label: () => t("nav.export"), icon: () => h(NIcon, null, "📤") },
  { key: "/organize", label: () => t("nav.organize"), icon: () => h(NIcon, null, "📁") },
  { key: "/settings", label: () => t("nav.settings"), icon: () => h(NIcon, null, "⚙️") },
  { key: "/about", label: () => t("nav.about"), icon: () => h(NIcon, null, "ℹ️") },
]
const activeKey = computed(() => route.path)
function onMenuChange(key: string) { router.push(key) }
function switchLang(locale: "zh-CN" | "en") {
  setLocale(locale)
  settingsStore.updateSettings({ locale } as any).catch(() => {})
}
</script>

<style>
.n-menu * { font-style: normal !important; }
</style>
