<template>
  <!-- 右下角浮动按钮（收起态） -->
  <n-button
    v-if="!player.isVisible.value"
    circle
    :style="{ position: 'fixed', bottom: '20px', right: '20px', zIndex: 1000, fontSize: '20px' }"
    @click="player.show()"
    title="音频播放器"
  >
    🎵
  </n-button>
  <n-card
    v-else
    :style="{ position: 'fixed', bottom: '16px', right: '16px', zIndex: 1000, width: '320px' }"
    size="small"
    :bordered="true"
  >
    <template #header>
      <n-space align="center" justify="space-between">
        <n-ellipsis :style="{ maxWidth: '240px' }">
          <n-text depth="2" style="font-size:13px;">
            {{ player.currentTitle.value || player.currentFile.value || '音频播放器' }}
          </n-text>
        </n-ellipsis>
        <n-button text size="tiny" @click="player.hide()">✕</n-button>
      </n-space>
    </template>
    <audio
      ref="audioRef"
      :src="audioSrc"
      controls
      style="width:100%;height:36px;display:block;"
    />
  </n-card>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { NButton, NCard, NSpace, NText, NEllipsis } from "naive-ui"
import { useAudioPlayer } from "../../composables/useAudioPlayer"

const player = useAudioPlayer()
const audioRef = ref<HTMLAudioElement | null>(null)

const audioSrc = computed(() => {
  if (!player.currentFile.value) return ""
  return `/api/files/preview?file=${encodeURIComponent(player.currentFile.value)}`
})

watch(() => player.currentFile.value, (newFile, oldFile) => {
  if (newFile && newFile !== oldFile && audioRef.value) {
    audioRef.value.src = audioSrc.value
    audioRef.value.play().catch(() => {})
  }
})

watch(() => player.isPlaying.value, (playing) => {
  if (!audioRef.value) return
  if (playing && audioRef.value.paused) {
    audioRef.value.play().catch(() => {})
  } else if (!playing && !audioRef.value.paused) {
    audioRef.value.pause()
  }
})
</script>

