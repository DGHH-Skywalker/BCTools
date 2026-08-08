<template>
  <!-- 播放器卡片 -->
  <n-card
    v-if="player.isVisible.value"
    :style="{ position: 'fixed', left: position.x + 'px', top: position.y + 'px', zIndex: 1000, width: '320px' }"
    size="small"
    :bordered="true"
  >
    <template #header>
      <div
        @mousedown="startDrag"
        @touchstart.prevent="startDrag"
        style="cursor:grab;display:flex;align-items:center;justify-content:space-between;"
      >
        <n-space align="center" style="min-width:0;flex:1;">
          <n-ellipsis style="max-width:240px;" :line-clamp="1">
            <n-text depth="2" style="font-size:13px;">
              {{ player.currentTitle.value || player.currentFile.value || '音频播放器' }}
            </n-text>
          </n-ellipsis>
        </n-space>
        <n-button text size="tiny" @click="player.hide()">
          <Close theme="outline" :size="14" :strokeWidth="3" />
        </n-button>
      </div>
    </template>

    <n-space vertical style="width:100%;">
      <n-text v-if="player.error.value" type="error" style="font-size:12px;">
        {{ player.error.value }}
      </n-text>

      <!-- 进度条 -->
      <n-slider
        :value="player.currentTime.value"
        :max="player.duration.value || 1"
        :step="1"
        :tooltip="false"
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
          <n-button size="small" :disabled="!player.currentFile.value" @click="stopAndClose">停止</n-button>
        </n-space>
      </n-space>
    </n-space>
  </n-card>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from "vue"
import { NEllipsis, NText } from "naive-ui"
import { Close } from "@icon-park/vue-next"
import { useAudioPlayer, formatDuration } from "../../composables/useAudioPlayer"

const player = useAudioPlayer()

const cardWidth = 320
const cardHeight = 160

function defaultPosition() {
  return {
    x: 16,
    y: window.innerHeight - cardHeight - 16,
  }
}

function positionFromButton(rect: { left: number; top: number; right: number; width: number; height: number }) {
  // 窗口放在按钮右侧，顶部与按钮顶部对齐，若超出屏幕则向左/上偏移
  let x = rect.right + 8
  let y = rect.top
  if (x + cardWidth > window.innerWidth) {
    x = rect.left - cardWidth - 8
  }
  if (y + cardHeight > window.innerHeight) {
    y = Math.max(0, window.innerHeight - cardHeight - 16)
  }
  return { x: Math.max(0, x), y: Math.max(0, y) }
}

const position = ref(defaultPosition())

const drag = ref<{ startX: number; startY: number; initialX: number; initialY: number } | null>(null)

function clampPosition(x: number, y: number) {
  return {
    x: Math.max(0, Math.min(x, window.innerWidth - cardWidth)),
    y: Math.max(0, Math.min(y, window.innerHeight - cardHeight)),
  }
}

function startDrag(e: MouseEvent | TouchEvent) {
  const clientX = "touches" in e ? e.touches[0].clientX : e.clientX
  const clientY = "touches" in e ? e.touches[0].clientY : e.clientY
  drag.value = {
    startX: clientX,
    startY: clientY,
    initialX: position.value.x,
    initialY: position.value.y,
  }
  window.addEventListener("mousemove", onDrag)
  window.addEventListener("mouseup", stopDrag)
  window.addEventListener("touchmove", onDrag)
  window.addEventListener("touchend", stopDrag)
}

function onDrag(e: MouseEvent | TouchEvent) {
  if (!drag.value) return
  const clientX = "touches" in e ? e.touches[0].clientX : e.clientX
  const clientY = "touches" in e ? e.touches[0].clientY : e.clientY
  const x = drag.value.initialX + (clientX - drag.value.startX)
  const y = drag.value.initialY + (clientY - drag.value.startY)
  position.value = clampPosition(x, y)
}

function stopDrag() {
  drag.value = null
  window.removeEventListener("mousemove", onDrag)
  window.removeEventListener("mouseup", stopDrag)
  window.removeEventListener("touchmove", onDrag)
  window.removeEventListener("touchend", stopDrag)
}

function onResize() {
  position.value = clampPosition(position.value.x, position.value.y)
}

onMounted(() => {
  window.addEventListener("resize", onResize)
  watch(
    () => player.initialPosition.value,
    (pos) => {
      if (pos) {
        position.value = clampPosition(pos.x, pos.y)
      }
    },
    { immediate: true },
  )
})

onUnmounted(() => {
  window.removeEventListener("resize", onResize)
  stopDrag()
})

function togglePlay() {
  if (player.isPlaying.value) {
    player.pause()
  } else {
    player.resume()
  }
}

// 点击停止按钮：停止播放并直接关闭组件
function stopAndClose() {
  player.stop()
  player.hide()
}
</script>

<style scoped>
/* 修复 n-ellipsis 悬停 tooltip 在亮色模式下文字变黑的问题 */
:deep(.n-ellipsis__tooltip),
:deep(.n-ellipsis__tooltip .n-popover__content),
:deep(.n-ellipsis__tooltip .n-popover__content *) {
  color: #fff !important;
}
:deep(.n-ellipsis__tooltip .n-popover__content) {
  background-color: rgba(0, 0, 0, 0.85) !important;
}
</style>
