<template>
  <div
    v-if="player.isVisible.value"
    ref="wrapperRef"
    class="audio-player-wrapper"
    :class="{ 'is-closing': isClosing }"
    :style="wrapperStyle"
  >
    <n-card size="small" :bordered="true" style="width:100%;">
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
          <n-button text size="tiny" @click="closeAnimated">
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from "vue"
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
const wrapperRef = ref<HTMLElement | null>(null)
const isClosing = ref(false)
const transitionEnabled = ref(false)
const transformStyle = ref({
  transform: "translate(0,0) scale(1)",
  opacity: 1,
  transformOrigin: "top left" as const,
})

const wrapperStyle = computed(() => ({
  position: "fixed" as const,
  left: `${position.value.x}px`,
  top: `${position.value.y}px`,
  width: `${cardWidth}px`,
  zIndex: 1000,
  transform: transformStyle.value.transform,
  opacity: transformStyle.value.opacity,
  transformOrigin: transformStyle.value.transformOrigin,
  transition: transitionEnabled.value ? "transform 250ms ease, opacity 250ms ease" : "none",
}))

function clampPosition(x: number, y: number) {
  return {
    x: Math.max(0, Math.min(x, window.innerWidth - cardWidth)),
    y: Math.max(0, Math.min(y, window.innerHeight - cardHeight)),
  }
}

function computeOriginTransform(origin: DOMRect) {
  return {
    transform: `translate(${origin.left - position.value.x}px, ${origin.top - position.value.y}px) scale(0.2)`,
    opacity: 0,
    transformOrigin: "top left" as const,
  }
}

function animateOpen() {
  const origin = player.originRect.value
  if (!origin) return
  transitionEnabled.value = false
  transformStyle.value = computeOriginTransform(origin)
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      transitionEnabled.value = true
      transformStyle.value = {
        transform: "translate(0,0) scale(1)",
        opacity: 1,
        transformOrigin: "top left",
      }
    })
  })
}

function closeAnimated() {
  const origin = player.originRect.value
  if (!origin) {
    player.hide()
    return
  }
  isClosing.value = true
  transitionEnabled.value = true
  transformStyle.value = computeOriginTransform(origin)
  window.setTimeout(() => {
    player.hide()
    isClosing.value = false
  }, 260)
}

const drag = ref<{ startX: number; startY: number; initialX: number; initialY: number } | null>(null)

function startDrag(e: MouseEvent | TouchEvent) {
  if (isClosing.value) return
  transitionEnabled.value = false
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
  // 若从按钮位置展开，播放进入动画
  if (player.originRect.value) {
    animateOpen()
  }
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
  closeAnimated()
}
</script>

<style scoped>
.audio-player-wrapper {
  will-change: transform, opacity;
}

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
