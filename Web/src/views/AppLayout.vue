<template>
  <n-layout style="height:100vh;" has-sider>
    <n-layout-sider v-if="!isHome && !isMobile" bordered :collapsed-width="64" :width="200" show-trigger="bar" collapse-mode="width" :collapsed="collapsed" @collapse="collapsed=true" @expand="collapsed=false">
      <div class="sider-menu-wrapper">
        <n-menu :collapsed="collapsed" :collapsed-width="64" :collapsed-icon-size="22" :value="activeKey" :options="visibleMenuOptions" @update:value="onMenuChange" />
      </div>
    </n-layout-sider>
    <n-layout style="height:100vh;">
      <n-layout-header v-if="config.languageSwitchEnabled || (isMobile && !isHome)" style="padding:8px 16px;background:#fff;border-bottom:1px solid #e0e0e0;display:flex;align-items:center;height:49px;">
        <n-button v-if="isMobile && !isHome" text style="margin-right:auto;" @click="mobileMenuOpen = true">
          <template #icon>
            <HamburgerButton theme="outline" :size="20" :strokeWidth="3" />
          </template>
          {{ t("common.menu") }}
        </n-button>
        <n-button-group v-if="config.languageSwitchEnabled" size="small" style="margin-left:auto;">
          <n-button :type="currentLocale==='zh-CN'?'primary':'default'" @click="switchLang('zh-CN')">中</n-button>
          <n-button :type="currentLocale==='en'?'primary':'default'" @click="switchLang('en')">EN</n-button>
        </n-button-group>
      </n-layout-header>
      <n-layout-content
        :native-scrollbar="true"
        :scrollbar-props="{ trigger: 'none' }"
        :style="{ padding: 0, height: (config.languageSwitchEnabled || (isMobile && !isHome)) ? 'calc(100vh - 49px)' : '100vh' }"
      >
        <router-view />
      </n-layout-content>
    </n-layout>

    <n-drawer v-if="isMobile && !isHome" v-model:show="mobileMenuOpen" placement="left" :width="260" :native-scrollbar="false">
      <n-space vertical style="padding:16px;">
        <n-menu :value="activeKey" :options="visibleMenuOptions" @update:value="onMobileMenuChange" />
      </n-space>
    </n-drawer>

    <!-- 音频预览：左下角浮动按钮（桌面端），仅当存在激活音频且播放器隐藏时显示 -->
    <n-button
      v-if="!isMobile && player.currentFile.value && !player.isVisible.value"
      circle
      type="primary"
      :title="t('nav.audioPreview')"
      class="audio-preview-fab"
      @click="showAudioPreview"
    >
      <template #icon>
        <Music theme="outline" :size="22" :strokeWidth="3" />
      </template>
    </n-button>
  </n-layout>
</template>
<script setup lang="ts">
import { ref, computed, h, onMounted, onUnmounted } from "vue"
import { useRouter, useRoute } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useAppConfig } from "../composables/useAppConfig"
import { useAudioPlayer } from "../composables/useAudioPlayer"
import type { MenuOption } from "naive-ui"
import { Clipboard, Broadcast, Export, FolderOpen, Setting, HamburgerButton, Music, Phone, Key } from "@icon-park/vue-next"

const router = useRouter(); const route = useRoute()
const { t, setLocale, currentLocale } = useI18n()
const settingsStore = useSettingsStore()
const { config } = useAppConfig()
const player = useAudioPlayer()
const collapsed = ref(true)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)
function updateIsMobile() { isMobile.value = window.innerWidth <= 768 }
onMounted(() => { updateIsMobile(); window.addEventListener("resize", updateIsMobile) })
onUnmounted(() => { window.removeEventListener("resize", updateIsMobile) })

const iconProps = { theme: "outline" as const, size: 20, strokeWidth: 3 }
const routeMenuOptions: MenuOption[] = [
  { key: "/dorm/manage", label: () => t("nav.dormManage"), icon: () => h(Clipboard, iconProps as any) },
  { key: "/broadcast", label: () => t("nav.broadcast"), icon: () => h(Broadcast, iconProps as any) },
  { key: "/export", label: () => t("nav.export"), icon: () => h(Export, iconProps as any) },
  { key: "/organize", label: () => t("nav.organize"), icon: () => h(FolderOpen, iconProps as any) },
  { key: "/decrypt", label: () => t("nav.decrypt"), icon: () => h(Key, iconProps as any) },
  { key: "/settings", label: () => t("nav.settings"), icon: () => h(Setting, iconProps as any) },
  { key: "/mobile", label: () => t("nav.mobile"), icon: () => h(Phone, iconProps as any) },
]
const menuOptions = computed((): MenuOption[] => routeMenuOptions)

const visibleMenuOptions = computed((): MenuOption[] => menuOptions.value)
const activeKey = computed(() => route.path)
const isHome = computed(() => route.path === "/home")
function onMenuChange(key: string) {
  router.push(key)
}
function onMobileMenuChange(key: string) {
  mobileMenuOpen.value = false
  router.push(key)
}
function showAudioPreview(e: MouseEvent) {
  const target = e.currentTarget as HTMLElement | null
  if (target) {
    player.show(target.getBoundingClientRect())
  } else {
    player.show()
  }
}
function switchLang(locale: "zh-CN" | "en") {
  setLocale(locale)
  settingsStore.updateSettings({ locale } as any).catch(() => {})
}
</script>

<style>
.n-menu * { font-style: normal !important; }
.sider-menu-wrapper {
  display: flex;
  flex-direction: column;
  justify-content: center;
  height: 100%;
}
.audio-preview-fab {
  position: fixed;
  left: 24px;
  bottom: 24px;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}
</style>
