import { ref } from "vue"

const audio = new Audio()
const currentFile = ref<string | null>(null)

audio.addEventListener("ended", () => { currentFile.value = null })
audio.addEventListener("pause", () => { currentFile.value = null })

export function useAudioPreview() {
  function play(filePath: string) {
    if (!filePath) return
    const url = `/api/files/preview?file=${encodeURIComponent(filePath)}`
    if (currentFile.value === filePath && !audio.paused) {
      audio.pause()
      currentFile.value = null
      return
    }
    audio.src = url
    audio.play().catch(() => {})
    currentFile.value = filePath
  }
  return { play, currentFile }
}
