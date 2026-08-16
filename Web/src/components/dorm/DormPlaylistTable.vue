<template>
  <div v-html="tableHtml" />
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "../../i18n"
import { buildDormTableHTML, ensureSegmenter, isSegmenterReady } from "../../utils/playlistTable"
import type { Song, TimeSlot } from "../../api/types"

const props = defineProps<{
  dates: string[]
  songs: Song[]
  timeSlots: TimeSlot[]
  title?: string | null
  emptyTitleText?: string
}>()

const { t, weekdayShortName } = useI18n()

// 分词词典是动态加载的独立块。先按不分词渲染（内容完全正确，只是少了按词断行
// 的零宽空格），词典到位后翻转该标志触发重算。
const segmenterReady = ref(isSegmenterReady())

onMounted(async () => {
  if (segmenterReady.value) return
  await ensureSegmenter()
  segmenterReady.value = isSegmenterReady()
})

const tableHtml = computed(() => {
  // 显式读取，让 computed 依赖词典就绪状态。
  void segmenterReady.value
  return buildDormTableHTML(props.dates, {
    title: props.title ?? null,
    songs: props.songs,
    timeSlots: props.timeSlots,
    weekdayShortName,
    emptyTitleText: props.emptyTitleText,
  })
})
</script>
