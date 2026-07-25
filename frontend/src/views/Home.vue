<template>
  <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;padding:60px 20px;min-height:calc(100vh - 60px);">
    <n-h1>{{ t("home.title") }}</n-h1>
    <n-p style="font-size:16px;color:#666;margin-bottom:24px;text-align:center;">{{ t("home.subtitle") }}</n-p>
    <n-button type="primary" size="large" @click="$router.push('/song/import')">{{ t("home.startButton") }}</n-button>
    <n-steps v-if="showGuide" vertical style="margin-top:40px;max-width:400px;">
      <n-step :title="t('home.step1')" />
      <n-step :title="t('home.step2')" />
      <n-step :title="t('home.step3')" />
    </n-steps>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useSongsStore } from "../stores/songs"
const { t } = useI18n()
const settingsStore = useSettingsStore()
const songsStore = useSongsStore()
const showGuide = ref(false)
onMounted(async () => {
  try {
    await Promise.all([settingsStore.fetchSettings(), songsStore.fetchSongs("dorm"), songsStore.fetchSongs("broadcast")])
    const isEmpty = settingsStore.timeSlots.length > 0 && songsStore.dormSongs.length === 0 && songsStore.broadcastSongs.length === 0
    showGuide.value = isEmpty
  } catch { showGuide.value = true }
})
</script>
