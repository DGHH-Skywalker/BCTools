import { ref, readonly } from "vue"

// ---- 模块级全局单例状态 ----
const currentFile = ref<string | null>(null)
const currentTitle = ref<string>("")
const isPlaying = ref(false)
const isVisible = ref(false)
const currentTime = ref(0)
const duration = ref(0)

const audio = new Audio()
audio.preload = "metadata"

audio.addEventListener("ended", () => {
  isPlaying.value = false
  currentTime.value = 0
})

audio.addEventListener("pause", () => {
  isPlaying.value = false
})

audio.addEventListener("play", () => {
  isPlaying.value = true
})

audio.addEventListener("timeupdate", () => {
  currentTime.value = audio.currentTime || 0
})

audio.addEventListener("loadedmetadata", () => {
  duration.value = audio.duration || 0
})

audio.addEventListener("emptied", () => {
  duration.value = 0
  currentTime.value = 0
})

export function useAudioPlayer() {
  function play(filePath: string, title = "") {
    if (!filePath) return

    const url = `/api/files/preview?file=${encodeURIComponent(filePath)}`

    // 同一首歌：切换播放/暂停
    if (currentFile.value === filePath) {
      if (audio.paused) {
        audio.play().catch(() => {})
      } else {
        audio.pause()
      }
      return
    }

    // 切换新歌：先停止当前，再播放新音频，保证全局只有一个声音
    audio.pause()
    audio.src = url
    audio.load()
    audio.play().catch(() => {})
    currentFile.value = filePath
    currentTitle.value = title
    isVisible.value = true
  }

  function pause() {
    audio.pause()
  }

  function resume() {
    if (currentFile.value && audio.paused) {
      audio.play().catch(() => {})
    }
  }

  function stop() {
    audio.pause()
    audio.src = ""
    currentFile.value = null
    currentTitle.value = ""
    isPlaying.value = false
    currentTime.value = 0
    duration.value = 0
  }

  function seek(time: number) {
    if (!isFinite(time)) return
    audio.currentTime = Math.max(0, Math.min(time, duration.value || time))
  }

  function hide() {
    pause()
    isVisible.value = false
  }

  function show() {
    isVisible.value = true
  }

  function toggle() {
    if (isVisible.value) hide()
    else show()
  }

  return {
    currentFile: readonly(currentFile),
    currentTitle: readonly(currentTitle),
    isPlaying: readonly(isPlaying),
    isVisible: readonly(isVisible),
    currentTime: readonly(currentTime),
    duration: readonly(duration),
    audioElement: audio,
    play,
    pause,
    resume,
    stop,
    seek,
    hide,
    show,
    toggle,
  }
}

export function formatDuration(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) return "00:00"
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`
}
