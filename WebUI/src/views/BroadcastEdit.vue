<template>
  <div>
    <PageTitle :title="t('broadcast.title')" />
    <BroadcastGrid />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
import BroadcastGrid from "../components/broadcast/BroadcastGrid.vue"
import PageTitle from "../components/common/PageTitle.vue"
const { t } = useI18n()
</script>

