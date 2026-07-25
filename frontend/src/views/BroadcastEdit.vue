<template>
  <SongTable type="broadcast" :title="t('broadcast.title')" />
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
import SongTable from "../components/common/SongTable.vue"
const { t } = useI18n()
</script>
