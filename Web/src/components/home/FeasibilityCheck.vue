<template>
  <div class="feasibility-check">
    <n-h2 style="text-align:center;font-family:var(--font-title);color:var(--theme-color);">{{ t("feasibilityCheck.title") }}</n-h2>

    <div class="check-row-list">
      <n-card v-for="item in items" :key="item.id" class="check-card" size="small">
        <div class="check-card-inner">
          <n-spin v-if="item.status === 'running'" size="small" />
          <check-one v-else-if="item.status === 'pass'" class="check-icon check-icon-pass" theme="outline" :size="22" fill="#27ae60" />
          <close-one v-else-if="item.status === 'fail'" class="check-icon check-icon-fail" theme="outline" :size="22" fill="#e74c3c" />
          <n-icon v-else :size="22"><more-one /></n-icon>
          <div class="check-text">
            <n-text strong class="check-name">{{ item.name }}</n-text>
            <n-text depth="3" class="check-status" :class="{ 'check-status-fail': item.status === 'fail' }">{{ statusText(item.status) }}</n-text>
          </div>
        </div>
      </n-card>
    </div>

    <n-p class="disclaimer">{{ t("feasibilityCheck.disclaimer") }}</n-p>

    <n-space justify="center" style="margin-top:16px;">
      <n-button size="large" @click="$emit('back')">{{ t("feasibilityCheck.back") }}</n-button>
      <n-button size="large" :loading="checking" @click="runChecks">{{ t("feasibilityCheck.recheck") }}</n-button>
      <n-button type="primary" size="large" :loading="checking" @click="start">{{ t("feasibilityCheck.start") }}</n-button>
    </n-space>

    <n-modal
      v-model:show="showConfirm"
      preset="dialog"
      :title="t('feasibilityCheck.confirmTitle')"
      :content="t('feasibilityCheck.confirmContent')"
      :positive-text="t('feasibilityCheck.enterAnyway')"
      :negative-text="t('common.cancel')"
      @positive-click="$router.push('/dorm/manage')"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../../i18n"
import { getSystemStatus } from "../../api/misc"
import { CheckOne, CloseOne, MoreOne } from "@icon-park/vue-next"

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

// 检测项说明：
// - backend/ffmpeg/ffprobe 共用一次 /system/status 调用
// - fileSelect（永远通过）与 localhost（与局域网点歌场景冲突）已移除
const items = ref<CheckItem[]>([
  { id: "backend", name: t("feasibilityCheck.backend"), status: "idle" },
  { id: "ffmpeg", name: t("feasibilityCheck.ffmpeg"), status: "idle" },
  { id: "ffprobe", name: t("feasibilityCheck.ffprobe"), status: "idle" },
  { id: "localStorage", name: t("feasibilityCheck.localStorage"), status: "idle" },
  { id: "audio", name: t("feasibilityCheck.audio"), status: "idle" },
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

async function checkBackendAndBinaries(): Promise<{ backend: boolean; ffmpeg: boolean; ffprobe: boolean }> {
  try {
    const status = await getSystemStatus()
    return { backend: true, ffmpeg: status.ffmpegAvailable, ffprobe: status.ffprobeAvailable }
  } catch {
    return { backend: false, ffmpeg: false, ffprobe: false }
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

async function runChecks() {
  items.value.forEach((i) => { i.status = "running" })
  checking.value = true

  const [server, storageOk, audioOk] = await Promise.all([
    checkBackendAndBinaries(),
    Promise.resolve(checkLocalStorage()),
    Promise.resolve(checkAudio()),
  ])
  setStatus("backend", server.backend ? "pass" : "fail")
  setStatus("ffmpeg", server.ffmpeg ? "pass" : "fail")
  setStatus("ffprobe", server.ffprobe ? "pass" : "fail")
  setStatus("localStorage", storageOk ? "pass" : "fail")
  setStatus("audio", audioOk ? "pass" : "fail")

  checking.value = false
}

function start() {
  if (allPassed.value) {
    router.push("/dorm/manage")
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

/* 一行放下所有卡片：flex 布局，空间不足时才换行 */
.check-row-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 16px;
}

.check-card {
  flex: 1 1 0;
  min-width: 140px;
}

.check-card-inner {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
}

.check-icon {
  flex-shrink: 0;
}

.check-text {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
  overflow: hidden;
}

.check-name {
  font-size: 14px;
}

.check-status {
  font-size: 12px;
}

.check-status-fail {
  color: var(--color-error);
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

  .check-card {
    min-width: calc(50% - 6px);
  }
}
</style>
