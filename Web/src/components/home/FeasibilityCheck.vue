<template>
  <div class="feasibility-check">
    <n-h2 style="text-align:center;font-family:var(--font-title);color:var(--theme-color);">{{ t("feasibilityCheck.title") }}</n-h2>

    <n-grid cols="2 s:3 m:4" :x-gap="12" :y-gap="12" style="margin-top:16px;">
      <n-grid-item v-for="item in items" :key="item.id">
        <n-card class="check-card" size="small">
          <n-space vertical align="center" style="text-align:center;">
            <n-spin v-if="item.status === 'running'" size="small" />
            <check-one v-else-if="item.status === 'pass'" theme="outline" :size="28" fill="#52c41a" />
            <close-one v-else-if="item.status === 'fail'" theme="outline" :size="28" fill="#ff4d4f" />
            <n-text strong>{{ item.name }}</n-text>
            <n-text depth="3">{{ statusText(item.status) }}</n-text>
          </n-space>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-p class="disclaimer">{{ t("feasibilityCheck.disclaimer") }}</n-p>

    <n-space justify="center" style="margin-top:16px;">
      <n-button size="large" @click="$emit('back')">{{ t("feasibilityCheck.back") }}</n-button>
      <n-button type="primary" size="large" :loading="checking" @click="start">{{ t("feasibilityCheck.start") }}</n-button>
    </n-space>

    <n-modal
      v-model:show="showConfirm"
      preset="dialog"
      :title="t('feasibilityCheck.confirmTitle')"
      :content="t('feasibilityCheck.confirmContent')"
      :positive-text="t('feasibilityCheck.enterAnyway')"
      :negative-text="t('common.cancel')"
      @positive-click="$router.push('/song/import')"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../../i18n"
import { getSystemStatus } from "../../api/system"
import { CheckOne, CloseOne } from "@icon-park/vue-next"

type CheckStatus = "idle" | "running" | "pass" | "fail"

interface CheckItem {
  id: string
  name: string
  status: CheckStatus
}

const { t } = useI18n()
const router = useRouter()
const checking = ref(false)
const showConfirm = ref(false)

const items = ref<CheckItem[]>([
  { id: "backend", name: t("feasibilityCheck.backend"), status: "idle" },
  { id: "ffmpeg", name: t("feasibilityCheck.ffmpeg"), status: "idle" },
  { id: "ffprobe", name: t("feasibilityCheck.ffprobe"), status: "idle" },
  { id: "localStorage", name: t("feasibilityCheck.localStorage"), status: "idle" },
  { id: "audio", name: t("feasibilityCheck.audio"), status: "idle" },
  { id: "fileSelect", name: t("feasibilityCheck.fileSelect"), status: "idle" },
  { id: "localhost", name: t("feasibilityCheck.localhost"), status: "idle" },
])

const allPassed = computed(() => items.value.every((i) => i.status === "pass"))

function statusText(status: CheckStatus) {
  if (status === "running") return t("feasibilityCheck.checking")
  if (status === "pass") return t("feasibilityCheck.pass")
  if (status === "fail") return t("feasibilityCheck.fail")
  return ""
}

function setStatus(id: string, status: CheckStatus) {
  const item = items.value.find((i) => i.id === id)
  if (item) item.status = status
}

async function checkBackend(): Promise<boolean> {
  try {
    await getSystemStatus()
    return true
  } catch {
    return false
  }
}

async function checkSystemBinaries(): Promise<{ ffmpeg: boolean; ffprobe: boolean }> {
  try {
    const status = await getSystemStatus()
    return { ffmpeg: status.ffmpegAvailable, ffprobe: status.ffprobeAvailable }
  } catch {
    return { ffmpeg: false, ffprobe: false }
  }
}

function checkLocalStorage(): boolean {
  try {
    const key = "__xb_storage_test__"
    localStorage.setItem(key, "1")
    localStorage.removeItem(key)
    return true
  } catch {
    return false
  }
}

function checkAudio(): boolean {
  try {
    const a = document.createElement("audio")
    return !!(a.canPlayType && a.canPlayType("audio/mpeg") !== "")
  } catch {
    return false
  }
}

function checkFileSelect(): boolean {
  const input = document.createElement("input")
  input.type = "file"
  return input.type === "file"
}

function checkLocalhost(): boolean {
  const host = window.location.hostname
  return host === "localhost" || host === "127.0.0.1" || host === "::1"
}

async function runChecks() {
  items.value.forEach((i) => { i.status = "running" })
  checking.value = true

  const [backendOk, binaryOk] = await Promise.all([
    checkBackend(),
    checkSystemBinaries(),
  ])
  setStatus("backend", backendOk ? "pass" : "fail")
  setStatus("ffmpeg", binaryOk.ffmpeg ? "pass" : "fail")
  setStatus("ffprobe", binaryOk.ffprobe ? "pass" : "fail")

  const others = await Promise.all([
    checkLocalStorage(),
    checkAudio(),
    checkFileSelect(),
    checkLocalhost(),
  ])
  setStatus("localStorage", others[0] ? "pass" : "fail")
  setStatus("audio", others[1] ? "pass" : "fail")
  setStatus("fileSelect", others[2] ? "pass" : "fail")
  setStatus("localhost", others[3] ? "pass" : "fail")

  checking.value = false
}

function start() {
  if (allPassed.value) {
    router.push("/song/import")
  } else {
    showConfirm.value = true
  }
}

defineEmits<{
  (e: "back"): void
}>()

onMounted(() => {
  runChecks()
})
</script>

<style scoped>
.feasibility-check {
  width: 100%;
  max-width: 900px;
  margin: 0 auto;
  padding: 16px;
  box-sizing: border-box;
}

.check-card {
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.disclaimer {
  margin-top: 16px;
  text-align: center;
  color: #888;
  font-size: 13px;
  line-height: 1.5;
}

@media (max-width: 480px) {
  .feasibility-check {
    padding: 8px;
  }
}
</style>
