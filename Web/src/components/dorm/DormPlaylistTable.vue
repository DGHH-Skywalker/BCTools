<template>
  <div v-html="tableHtml" />
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "../../i18n"
import { buildDormTableHTML } from "../../utils/playlistTable"
import type { Song, TimeSlot } from "../../api/types"

const props = defineProps<{
  dates: string[]
  songs: Song[]
  timeSlots: TimeSlot[]
  title?: string | null
  emptyTitleText?: string
}>()

const { t, weekdayShortName } = useI18n()

const tableHtml = computed(() => {
  return buildDormTableHTML(props.dates, {
    title: props.title ?? null,
    songs: props.songs,
    timeSlots: props.timeSlots,
    weekdayShortName,
    emptyTitleText: props.emptyTitleText,
  })
})
</script>
