<template>
  <div style="display:flex;align-items:center;justify-content:center;padding:60px 20px;min-height:calc(100vh - 60px);">
    <n-button type="primary" size="large" @click="$router.push('/song/import')">{{ t("home.startButton") }}</n-button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
const { t } = useI18n()
</script>
