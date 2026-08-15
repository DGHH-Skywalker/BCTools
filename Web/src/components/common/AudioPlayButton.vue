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
    size?: "tiny" | "small" | "medium" | "large"
  }>(),
  {
    size: "tiny",
  },
)

const { t } = useI18n()
const player = useAudioPlayer()
const buttonRef = ref<HTMLElement | null>(null)

const isCurrent = computed(() => player.currentFile.value === props.filePath)

function handleClick() {
  const rect = (buttonRef.value as any)?.$el?.getBoundingClientRect()
  if (rect && (!player.isVisible.value || !isCurrent.value)) {
    player.show(rect)
  }
  player.play(props.filePath, props.title)
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
