<template>
  <div></div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from "vue"
import { useNotification } from "naive-ui"

const notification = useNotification()
let intervalId: number | undefined
let watchAbort: AbortController | undefined
let offlineNotified = false
// 后端主动通知关闭后，不再弹「连接异常」——那是预期行为，不是故障。
let shuttingDown = false

async function ping() {
  if (shuttingDown) return
  try {
    const res = await fetch("/api/health", { cache: "no-store" })
    if (res.ok && offlineNotified) {
      offlineNotified = false
    }
  } catch {
    if (!offlineNotified && !shuttingDown) {
      offlineNotified = true
      notification.error({
        title: "服务端连接异常",
        content: "无法连接到后端服务，请检查程序是否仍在运行。",
        duration: 5000,
      })
    }
  }
}

// watchShutdown 挂一个长轮询在 /api/lifecycle/watch 上。托盘选择「退出程序」时
// 后端让它返回，页面随即尝试自行关闭。
//
// 浏览器普遍拦截脚本关闭非脚本打开的标签页，所以 window.close() 很可能无效。
// 关不掉时退化成一条提示，而不是留给用户一个连不上后端的死页面。
async function watchShutdown() {
  watchAbort = new AbortController()
  try {
    const res = await fetch("/api/lifecycle/watch", {
      cache: "no-store",
      signal: watchAbort.signal,
    })
    if (!res.ok) return
    shuttingDown = true
    if (intervalId) {
      window.clearInterval(intervalId)
      intervalId = undefined
    }
    window.close()
    // 走到这里说明浏览器拒绝了关闭请求。
    notification.info({
      title: "后端已退出",
      content: "程序已从托盘退出，可以关闭此页面了。",
      duration: 0,
    })
  } catch {
    // AbortError（组件卸载）或后端在返回前就断开，两种都无需处理。
  }
}

onMounted(() => {
  ping()
  intervalId = window.setInterval(ping, 30000)
  watchShutdown()
})

onUnmounted(() => {
  if (intervalId) window.clearInterval(intervalId)
  watchAbort?.abort()
})
</script>
