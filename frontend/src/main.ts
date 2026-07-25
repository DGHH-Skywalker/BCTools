import { createApp } from "vue"
import { createPinia } from "pinia"
import router from "./router"
import App from "./App.vue"
import { useErrorStore } from "./stores/error"

const app = createApp(App)
app.use(createPinia())
app.use(router)

// 全局错误处理
app.config.errorHandler = (err, _instance, info) => {
  const errorStore = useErrorStore()
  errorStore.setError(err)
  console.error("[Vue Error]", err, info)
  router.push("/error").catch(() => {})
}

app.mount("#app")
