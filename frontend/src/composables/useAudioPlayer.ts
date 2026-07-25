import { ref, readonly } from "vue"

// ---- 模块级全局单例状态 ----
const currentFile = ref<string | null>(null)
const currentTitle = ref<string>("")
const isPlaying = ref(false)
const isVisible = ref(false)

const audio = new Audio()

audio.addEventListener("ended", () => {
  isPlaying.value = false
  // 不自动清除 currentFile，让用户能看到播放完了的歌曲
})

audio.addEventListener("pause", () => {
  isPlaying.value = false
})

audio.addEventListener("play", () => {
  isPlaying.value = true
})

export function useAudioPlayer() {
  function play(filePath: string, title = "") {
    if (!filePath) return

    const url = `/api/files/preview?file=${encodeURIComponent(filePath)}`

    // 如果点击的是同一首歌
    if (currentFile.value === filePath) {
      if (audio.paused) {
        audio.play().catch(() => {})
      } else {
        audio.pause()
      }
      return
    }

    // 切换新歌
    audio.src = url
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
  }

  function hide() {
    pause()
    isVisible.value = false
  }

  function show() {
    isVisible.value = true
  }

  return {
    currentFile: readonly(currentFile),
    currentTitle: readonly(currentTitle),
    isPlaying: readonly(isPlaying),
    isVisible: readonly(isVisible),
    audioElement: audio,
    play,
    pause,
    resume,
    stop,
    hide,
    show,
  }
}
