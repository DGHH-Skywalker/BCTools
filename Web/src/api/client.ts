import axios from "axios"
import { activeMutations, reportSaveState } from "../utils/pendingSaves"

const apiClient = axios.create({
  baseURL: "/api",
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
})

const isMutation = (method?: string) => !["get", "head", "options"].includes((method || "get").toLowerCase())
apiClient.interceptors.request.use(async config => {
  if (isMutation(config.method) && !['/shutdown', '/close-browser'].includes(config.url || '')) {
    // Draft flush requests remain allowed; editing is frozen in the UI during handoff.
    activeMutations.value++
    ;(config as any).__trackedWrite = true
    try {
      const state = await reportSaveState()
      if (state.frozen) throw new Error("正在切换版本，请稍后重试")
      config.headers.set('X-BCTools-Instance', state.instanceId)
    } catch (error) {
      finishWrite(config)
      throw error
    }
  }
  return config
})
function finishWrite(config: any) {
  if (config?.__trackedWrite) { activeMutations.value--; config.__trackedWrite = false }
}

apiClient.interceptors.response.use(
  (res) => { finishWrite(res.config); return res },
  (error) => {
    finishWrite(error.config)
    if (!error.response) {
      console.error("[API] Network error:", error.message)
    } else {
      const { code, message } = error.response.data?.error ?? {}
      console.error(`[API] ${code}: ${message}`)
    }
    return Promise.reject(error)
  },
)

export default apiClient
