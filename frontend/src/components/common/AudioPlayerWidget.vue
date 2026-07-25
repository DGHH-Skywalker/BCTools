<template>
  <!-- 右下角浮动按钮 -->
  <n-float-button
    v-if="!player.isVisible.value"
    :right="20"
    :bottom="20"
    @click="player.show()"
    title="音频播放器"
  >
    🎵
  </n-float-button>

  <!-- 播放器卡片 -->
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

    <n-space vertical style="width:100%;">
      <!-- 进度条 -->
      <n-slider
        :value="player.currentTime.value"
        :max="player.duration.value || 1"
        :step="1"
        :disabled="!player.duration.value"
        @update:value="player.seek"
      />

      <n-space align="center" justify="space-between" style="width:100%;">
        <n-text depth="3" style="font-size:12px;">
          {{ formatDuration(player.currentTime.value) }} / {{ formatDuration(player.duration.value) }}
        </n-text>

        <n-space>
          <n-button
            size="small"
            :disabled="!player.currentFile.value"
            @click="togglePlay"
          >
            {{ player.isPlaying.value ? '暂停' : '播放' }}
          </n-button>
          <n-button size="small" :disabled="!player.currentFile.value" @click="player.stop()">停止</n-button>
        </n-space>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { NEllipsis, NText } from "naive-ui"
import { useAudioPlayer, formatDuration } from "../../composables/useAudioPlayer"

const player = useAudioPlayer()

function togglePlay() {
  if (player.isPlaying.value) {
    player.pause()
  } else {
    player.resume()
  }
}
</script>
