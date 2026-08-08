<template>
  <PageContainer>
    <n-grid cols="1 s:1 m:2 l:2 xl:2 xxl:2" :x-gap="16" :y-gap="16" class="settings-grid">
      <n-grid-item>
        <n-card>
          <n-space align="start" :wrap="false">
            <n-image :src="logoSrc" class="settings-logo" preview-disabled :img-props="{ draggable: false }" />
            <n-space vertical>
              <n-h1 class="settings-app-title">
                {{ t("app.title") }}
              </n-h1>
              <n-text>{{ t("about.softwareIntro") }}</n-text>
              <n-text depth="3">{{ t("settings.version") }}: {{ settingsStore.version }}</n-text>
              <n-button size="small" @click="goToGuide">
                <template #icon><Help theme="outline" :size="14" :strokeWidth="3" /></template>
                {{ t("about.guideButton") }}
              </n-button>
            </n-space>
          </n-space>
        </n-card>
      </n-grid-item>

      <n-grid-item>
        <n-card>
          <n-space align="start" :wrap="false">
            <n-image :src="authorAvatarSrc" class="settings-avatar" preview-disabled :img-props="{ draggable: false }" />
            <n-space vertical>
              <n-h2 class="settings-author-title">{{ t("about.authorTitle") }}</n-h2>
              <n-text strong>{{ t("about.author") }}</n-text>
              <n-text>{{ t("about.authorIntro") }}</n-text>
              <n-button text tag="a" href="https://github.com/Egansama" target="_blank">{{ t("about.github") }}</n-button>
            </n-space>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card class="settings-card">
      <n-space vertical size="large">
        <n-grid cols="1 s:2 m:3 l:3 xl:3" :x-gap="16" :y-gap="16">
          <n-grid-item v-if="isLocalhost">
            <n-space align="center">
              <n-text>{{ t("settings.autoStart") }}:</n-text>
              <n-switch v-model:value="settingsStore.autoStartEnabled" @update:value="saveAutoStartSetting" />
            </n-space>
          </n-grid-item>
        </n-grid>
      </n-space>
    </n-card>

  </PageContainer>
</template>

<script setup lang="ts">
import { onMounted, computed } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { Help } from "@icon-park/vue-next"
import PageContainer from "../components/ui/PageContainer.vue"

const { t } = useI18n()
const router = useRouter()
const settingsStore = useSettingsStore()

function goToGuide() {
  router.push("/guide")
}

const isLocalhost = computed(() => {
  const host = window.location.hostname
  return host === "localhost" || host === "127.0.0.1" || host === "::1"
})

const logoSrc = computed(() =>
  matchMedia("(prefers-color-scheme: dark)").matches ? "/logo-white.png" : "/logo.png"
)
const authorAvatarSrc = "/egansama.jpg"

onMounted(() => {
  settingsStore.fetchSettings()
})

async function saveAutoStartSetting(val: boolean) {
  await settingsStore.updateSettings({ autoStartEnabled: val } as any)
}
</script>

<style scoped>
.settings-grid {
  margin-bottom: var(--spacing-md);
}

.settings-logo {
  width: 96px;
  height: 96px;
  flex-shrink: 0;
}

.settings-avatar {
  width: 96px;
  height: 96px;
  flex-shrink: 0;
  object-fit: cover;
  border-radius: 50%;
}

.settings-app-title {
  margin: 0;
  font-family: var(--font-title);
  font-size: 2rem;
  color: var(--theme-color);
  white-space: nowrap;
}

.settings-author-title {
  margin: 0;
}

.settings-card {
  margin-bottom: var(--spacing-md);
}

</style>
