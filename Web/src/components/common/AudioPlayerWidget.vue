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
              :title="player.isPlaying.value ? '暂停' : '播放'"
              @click="togglePlay"
            >
              <template #icon>
                <Pause v-if="player.isPlaying.value" theme="outline" :size="14" :strokeWidth="3" />
                <Play v-else theme="outline" :size="14" :strokeWidth="3" />
              </template>
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
import { Close, Play, Pause } from "@icon-park/vue-next"
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
const closeTimeout = ref<number | null>(null)
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

function resetTransform() {
  transitionEnabled.value = false
  transformStyle.value = {
    transform: "translate(0,0) scale(1)",
    opacity: 1,
    transformOrigin: "top left",
  }
}

function animateOpen() {
  const origin = player.originRect.value
  if (!origin) {
    // 没有起点矩形就没有展开动画，但必须显式复位——否则会残留上一次
    // 收起动画留下的 scale(0.2)/opacity:0。
    resetTransform()
    return
  }
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
    player.close()
    return
  }
  isClosing.value = true
  transitionEnabled.value = true
  transformStyle.value = computeOriginTransform(origin)
  if (closeTimeout.value !== null) {
    window.clearTimeout(closeTimeout.value)
  }
  closeTimeout.value = window.setTimeout(() => {
    player.close()
    isClosing.value = false
    closeTimeout.value = null
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
  // 每次变为可见都要重跑展开动画，而不是只在挂载时跑一次。
  //
  // 该组件在 App.vue 里是常挂载的（外层 v-show），内部 wrapper 用 v-if 控制。
  // 关闭动画结束时 transform 停在 scale(0.2)/opacity:0；如果只在 onMounted
  // 里 animateOpen，第二次打开就会带着这份透明状态渲染出来——DOM 里有元素，
  // 但用户什么也看不见，表现为「按钮点了就消失，播放器再也打不开」。
  watch(
    () => player.isVisible.value,
    (visible) => {
      if (!visible) return
      // 收起动画可能还没跑完就又被打开，取消它以免 260ms 后把刚开的窗口关掉。
      if (closeTimeout.value !== null) {
        window.clearTimeout(closeTimeout.value)
        closeTimeout.value = null
      }
      isClosing.value = false
      animateOpen()
    },
    { immediate: true },
  )
})

onUnmounted(() => {
  window.removeEventListener("resize", onResize)
  stopDrag()
  if (closeTimeout.value !== null) {
    window.clearTimeout(closeTimeout.value)
    closeTimeout.value = null
  }
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
