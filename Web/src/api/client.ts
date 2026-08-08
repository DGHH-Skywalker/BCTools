import axios from "axios"

const apiClient = axios.create({
  baseURL: "/api",
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
})

apiClient.interceptors.response.use(
  (res) => res,
  (error) => {
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
