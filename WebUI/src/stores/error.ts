import { defineStore } from "pinia"
import { ref } from "vue"

export const useErrorStore = defineStore("error", () => {
  const message = ref<string | null>(null)
  const code = ref<string | null>(null)
  const status = ref<number | null>(null)
  const source = ref<string | null>(null)

  function setError(err: unknown) {
    if (err instanceof Error) {
      message.value = err.message
    } else if (typeof err === "string") {
      message.value = err
    } else {
      message.value = "未知错误"
    }
  }

  function clearError() {
    message.value = null
    code.value = null
    status.value = null
    source.value = null
  }

  return { message, code, status, source, setError, clearError }
})
