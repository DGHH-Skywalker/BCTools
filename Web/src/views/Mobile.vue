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
        <img v-if="networkInfo" class="mobile-qr" :src="`/api/network/qr?t=${qrTs}`" alt="App QR" />
        <n-text depth="3" class="mobile-tip">{{ t('mobile.sameNetworkTip') }}</n-text>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { getNetworkInfo } from "../api/network"
import type { NetworkInfo } from "../api/types"

const { t } = useI18n()
const router = useRouter()
const networkInfo = ref<NetworkInfo | null>(null)
const qrTs = ref(Date.now())
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

@media (max-width: 480px) {
  .mobile-qr {
    width: 160px;
    height: 160px;
  }
}
</style>
