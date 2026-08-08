<template>
  <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:100vh;">
    <n-h1>{{ t("error.title") }}</n-h1>
    <n-p>{{ errorStore.message || t("error.unknown") }}</n-p>
    <n-space>
      <n-button @click="$router.push('/home')">{{ t("error.goHome") }}</n-button>
      <n-button @click="reloadPage">{{ t("error.reload") }}</n-button>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
import { useErrorStore } from "../stores/error"
const { t } = useI18n()
const errorStore = useErrorStore()
function reloadPage() { window.location.reload() }
</script>
