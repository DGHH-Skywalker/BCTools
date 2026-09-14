<template>
  <n-button
    ref="buttonRef"
    :size="size"
    :type="isCurrent ? 'primary' : 'default'"
    :title="isCurrent && player.isPlaying.value ? t('common.pause') : t('common.listen')"
    @click="handleClick"
  >
    <template #icon>
      <Pause v-if="isCurrent && player.isPlaying.value" theme="outline" :size="iconSize" :strokeWidth="3" />
      <Play v-else theme="outline" :size="iconSize" :strokeWidth="3" />
    </template>
  </n-button>
</template>

<script setup lang="ts">
import { ref, computed } from "vue"
import { useI18n } from "../../i18n"
import { useAudioPlayer } from "../../composables/useAudioPlayer"
import { Play, Pause } from "@icon-park/vue-next"

const props = withDefaults(
  defineProps<{
    filePath: string
    title?: string
    // 5.6.0 起同一时段多首歌可能合并成一个文件，传 songId 时走
    // /api/files/stream?songId= 按歌单独播放（合并文件的成员也能单独试听）
    songId?: number
    size?: "tiny" | "small" | "medium" | "large"
  }>(),
  {
    size: "tiny",
  },
)

const { t } = useI18n()
const player = useAudioPlayer()
const buttonRef = ref<HTMLElement | null>(null)

// 与 useAudioPlayer.play 内部的播放身份保持一致
const playKey = computed(() => (props.songId != null ? `song:${props.songId}` : props.filePath))
const isCurrent = computed(() => player.currentFile.value === playKey.value)

function handleClick() {
  const rect = (buttonRef.value as any)?.$el?.getBoundingClientRect()
  if (rect && (!player.isVisible.value || !isCurrent.value)) {
    player.show(rect)
  }
  player.play(props.filePath, props.title, props.songId)
}

const iconSize = computed(() => {
  switch (props.size) {
    case "large":
      return 22
    case "medium":
      return 20
    case "small":
      return 16
    default:
      return 14
  }
})
</script>
