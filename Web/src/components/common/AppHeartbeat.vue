<template>
  <DraftRecovery :active-clients="activeClients" />
  <div v-if="unsavedCount || saveError || handoffFrozen" role="status" style="position:fixed;bottom:8px;left:16px;z-index:9999;background:#fff4d6;color:#502f00;padding:8px 12px;border-radius:6px;">
    {{ handoffFrozen ? '正在保存并切换版本，请稍候…' : (saveError || `正在保存 ${unsavedCount} 项编辑…`) }}
    <button v-if="saveError && !handoffFrozen" @click="retrySave">重试保存</button>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from "vue"
import { useNotification } from "naive-ui"
import { unsavedCount, saveError, handoffFrozen, activeMutations, browserClientID, flushAllSaves, reportSaveState } from "../../utils/pendingSaves"
import DraftRecovery from "./DraftRecovery.vue"

const notification = useNotification()
let intervalId: number | undefined
let watchAbort: AbortController | undefined
let offlineNotified = false
// 后端主动通知关闭后，不再弹「连接异常」——那是预期行为，不是故障。
let shuttingDown = false
let clientTimer: ReturnType<typeof setTimeout> | undefined
let stopped = false
let ack = ""
const activeClients = ref<string[]>([])

async function retrySave() {
  try { await flushAllSaves() } catch { /* error remains visible */ }
}

async function reportClient() {
  try {
    const state = await reportSaveState(ack)
    activeClients.value = state.clients || []
    if (saveError.value === '无法确认保存状态；请检查连接，暂勿关闭页面') saveError.value = ''
    if (state.prepare) {
      handoffFrozen.value = true
      document.querySelector<HTMLElement>('#app')?.setAttribute('inert', '')
      ;(document.activeElement as HTMLElement | null)?.blur()
      await flushAllSaves()
      ack = state.prepare
    } else {
      ack = ''
      handoffFrozen.value = false
      document.querySelector<HTMLElement>('#app')?.removeAttribute('inert')
    }
  } catch {
    // Never acknowledge a failed flush or discard an unsaved draft.
    ack = ''
    saveError.value ||= '无法确认保存状态；请检查连接，暂勿关闭页面'
  } finally {
    if (!stopped && !shuttingDown) clientTimer = setTimeout(reportClient, 1000)
  }
}

function beforeUnload(event: BeforeUnloadEvent) {
  if (unsavedCount.value || activeMutations.value) { event.preventDefault(); event.returnValue = '' }
}
function leave() {
  if (!unsavedCount.value && !activeMutations.value) {
    navigator.sendBeacon?.('/api/lifecycle/client', new Blob([JSON.stringify({ id: browserClientID, leave: true })], { type: 'application/json' }))
  }
}

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

watch([unsavedCount, activeMutations], () => {
  if (!stopped && !shuttingDown) void reportSaveState(ack).catch(() => { /* periodic poll handles recovery */ })
})

onMounted(() => {
  stopped = false
  document.querySelector<HTMLElement>('#app')?.setAttribute('inert', '')
  void reportClient()
  window.addEventListener('beforeunload', beforeUnload)
  window.addEventListener('pagehide', leave)
  ping()
  intervalId = window.setInterval(ping, 30000)
  watchShutdown()
})

onUnmounted(() => {
  stopped = true
  clearTimeout(clientTimer)
  leave()
  window.removeEventListener('beforeunload', beforeUnload)
  window.removeEventListener('pagehide', leave)
  if (intervalId) window.clearInterval(intervalId)
  watchAbort?.abort()
})
</script>
