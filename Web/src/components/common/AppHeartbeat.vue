<template>
  <div></div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from "vue"
import { useNotification } from "naive-ui"

const notification = useNotification()
let intervalId: number | undefined
let offlineNotified = false

async function ping() {
  try {
    const res = await fetch("/api/health", { cache: "no-store" })
    if (res.ok && offlineNotified) {
      offlineNotified = false
    }
  } catch {
    if (!offlineNotified) {
      offlineNotified = true
      notification.error({
        title: "服务端连接异常",
        content: "无法连接到后端服务，请检查程序是否仍在运行。",
        duration: 5000,
      })
    }
  }
}

onMounted(() => {
  ping()
  intervalId = window.setInterval(ping, 30000)
})

onUnmounted(() => {
  if (intervalId) window.clearInterval(intervalId)
})
</script>
