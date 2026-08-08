<template>
  <div v-if="showMobileWarning" class="mobile-warning">
    <n-image :src="logoSrc" class="mobile-warning-logo" preview-disabled :img-props="{ draggable: false }" />
    <n-h1 class="mobile-warning-title">{{ t("mobile.title") }}</n-h1>
    <n-p class="mobile-warning-text">{{ t("mobile.mobileWarning") }}</n-p>
    <n-space class="mobile-warning-actions">
      <n-button type="primary" @click="warningSkipped = true">{{ t("mobile.confirmEnter") }}</n-button>
      <n-button @click="router.back()">{{ t("mobile.goBack") }}</n-button>
    </n-space>
  </div>

  <div v-else class="mobile-content">
    <n-card class="mobile-card" :title="t('mobile.appUrlTitle')">
      <n-space vertical align="center">
        <n-text code class="mobile-url">{{ networkInfo?.url || '...' }}</n-text>
        <img v-if="networkInfo" class="mobile-qr" :src="`/api/network/qr?type=app&t=${qrTs}`" alt="App QR" />
        <n-text depth="3" class="mobile-tip">{{ t('mobile.sameNetworkTip') }}</n-text>
      </n-space>
    </n-card>

    <n-card class="mobile-card">
      <n-space justify="center">
        <n-button type="primary" @click="openHotspotModal">
          <template #icon>
            <Wifi theme="outline" />
          </template>
          {{ t('mobile.openHotspot') }}
        </n-button>
      </n-space>
    </n-card>

    <n-modal v-model:show="showModal" preset="card" :title="t('mobile.hotspotModalTitle')" :mask-closable="false" class="mobile-modal">
      <n-space vertical align="center" class="mobile-modal-body">
        <template v-if="hotspot.status === 'starting'">
          <n-spin size="large" />
          <n-text>{{ t('mobile.startingHotspot') }}</n-text>
        </template>
        <template v-else-if="hotspot.status === 'started'">
          <img class="mobile-qr" :src="`/api/network/qr?type=hotspot&t=${qrTs}`" alt="WiFi QR" />
          <n-space vertical class="mobile-hotspot-info">
            <n-space justify="space-between">
              <n-text strong>SSID</n-text>
              <n-text>{{ hotspot.ssid }}</n-text>
            </n-space>
            <n-space justify="space-between">
              <n-text strong>{{ t('mobile.password') }}</n-text>
              <n-text>{{ hotspot.password }}</n-text>
            </n-space>
          </n-space>
          <n-text depth="3">{{ t('mobile.hotspotReady') }}</n-text>
        </template>
        <template v-else-if="hotspot.status === 'failed'">
          <n-text type="error">{{ hotspot.message || t('mobile.hotspotFailed') }}</n-text>
          <n-button v-if="hotspot.fallback" @click="openSystemSettings">{{ t('mobile.openSystemSettings') }}</n-button>
        </template>
        <template v-else>
          <n-text depth="3">{{ t('mobile.hotspotIdle') }}</n-text>
        </template>
      </n-space>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { getNetworkInfo, startHotspot, getHotspotStatus } from "../api/network"
import type { NetworkInfo, HotspotStatus } from "../api/types"
import { Wifi } from "@icon-park/vue-next"

const { t } = useI18n()
const router = useRouter()
const networkInfo = ref<NetworkInfo | null>(null)
const hotspot = ref<HotspotStatus>({ status: "idle", ssid: "", password: "", message: "", fallback: "" })
const showModal = ref(false)
const qrTs = ref(Date.now())
let pollTimer: number | null = null
let refreshTimer: number | null = null

function isTouchDevice() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0
}
function isMobileUA() {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
}
const shouldWarn = isMobileUA() || (isTouchDevice() && !matchMedia("(pointer: fine)").matches)
const warningSkipped = ref(false)
const showMobileWarning = computed(() => shouldWarn && !warningSkipped.value)

const logoSrc = computed(() =>
  matchMedia("(prefers-color-scheme: dark)").matches ? "/logo-white.png" : "/logo.png"
)

async function refreshNetworkInfo() {
  try {
    networkInfo.value = await getNetworkInfo()
    qrTs.value = Date.now()
  } catch {
    // ignore
  }
}

onMounted(async () => {
  await refreshNetworkInfo()
  refreshTimer = window.setInterval(refreshNetworkInfo, 3000)
  window.addEventListener("visibilitychange", handleVisibilityChange)
})

onUnmounted(() => {
  stopPolling()
  if (refreshTimer !== null) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
  window.removeEventListener("visibilitychange", handleVisibilityChange)
})

function handleVisibilityChange() {
  if (document.visibilityState === "visible") {
    refreshNetworkInfo()
  }
}

watch(showModal, (visible) => {
  if (!visible) stopPolling()
})

function openHotspotModal() {
  showModal.value = true
  hotspot.value = { status: "starting", ssid: "", password: "", message: "", fallback: "" }
  startPolling()
  startHotspot().catch(() => {})
}

function startPolling() {
  stopPolling()
  pollTimer = window.setInterval(async () => {
    try {
      const status = await getHotspotStatus()
      hotspot.value = status
      if (status.status === "started" || status.status === "failed") {
        stopPolling()
      }
    } catch {
      // ignore
    }
  }, 500)
}

function stopPolling() {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function openSystemSettings() {
  if (hotspot.value.fallback) {
    window.open(hotspot.value.fallback, "_blank")
  }
}
</script>

<style scoped>
.mobile-warning {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--spacing-md);
  box-sizing: border-box;
}

.mobile-warning-logo {
  width: 120px;
  height: 120px;
}

.mobile-warning-title {
  margin-top: var(--spacing-lg);
  margin-bottom: 0;
}

.mobile-warning-text {
  max-width: 320px;
  text-align: center;
  margin-top: var(--spacing-sm);
}

.mobile-warning-actions {
  margin-top: var(--spacing-lg);
}

.mobile-content {
  padding: var(--spacing-md);
  max-width: 640px;
  margin: 0 auto;
}

.mobile-card {
  margin-bottom: var(--spacing-md);
}

.mobile-url {
  font-size: var(--spacing-md);
}

.mobile-qr {
  width: 200px;
  height: 200px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.mobile-tip {
  text-align: center;
}

.mobile-modal {
  width: 90%;
  max-width: 420px;
}

.mobile-modal-body {
  padding: var(--spacing-sm) 0;
}

.mobile-hotspot-info {
  width: 100%;
}

@media (max-width: 480px) {
  .mobile-qr {
    width: 160px;
    height: 160px;
  }
}
</style>
