<template>
  <n-space vertical :style="{ padding: '60px 20px', maxWidth: '500px', margin: '0 auto' }">
    <n-h1 style="text-align:center;">{{ t("app.title") }}</n-h1>
    <n-p style="text-align:center;">{{ t("about.description") }}</n-p>
    <n-divider />
    <n-space vertical size="large">
      <n-thing title="Egansama" description="Developer">
        <n-space vertical>
          <n-space align="center"><n-text depth="3">QQ:</n-text><n-text>{{ t("about.qq") }}</n-text></n-space>
          <n-space align="center"><n-text depth="3">GitHub:</n-text><n-a href="https://github.com/Egansama" target="_blank">{{ t("about.github") }}</n-a></n-space>
        </n-space>
      </n-thing>
    </n-space>
    <n-divider />
    <n-p style="text-align:center;">{{ t("about.copyright") }}</n-p>
  </n-space>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
const { t } = useI18n()
</script>
