<template>
  <div :style="{ padding: '60px 20px', maxWidth: '500px', margin: '0 auto' }">
    <PageTitle :title="t('app.title')" />
    <n-divider />
    <n-p style="text-align:center;">{{ t("about.copyright") }}</n-p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
import PageTitle from "../components/common/PageTitle.vue"
const { t } = useI18n()
</script>
