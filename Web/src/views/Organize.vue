<template>
  <div v-if="showMobileWarning" class="organize-mobile-warning">
    <n-image :src="logoSrc" class="organize-mobile-logo" preview-disabled :img-props="{ draggable: false }" />
    <n-h1 class="organize-mobile-title">{{ t("organize.title") }}</n-h1>
    <n-p class="organize-mobile-text">{{ t("organize.mobileWarning") }}</n-p>
    <n-space class="organize-mobile-actions">
      <n-button type="primary" @click="warningSkipped = true">{{ t("organize.confirmEnter") }}</n-button>
      <n-button @click="router.back()">{{ t("organize.goBack") }}</n-button>
    </n-space>
  </div>

  <div v-else class="organize-page">
    <n-card class="organize-card">
      <n-space vertical size="large">
        <n-space align="center" wrap>
          <n-text>{{ t("organize.targetDir") }}:</n-text>
          <n-input :value="targetDirName || ''" :placeholder="t('organize.noTargetDir')" class="organize-dir-input-field" disabled />
          <n-button @click="browse">{{ t("organize.browse") }}</n-button>
        </n-space>

        <n-space align="center" wrap>
          <n-text>{{ t("organize.year") }}:</n-text>
          <YearSelect v-model:value="selectedYear" />
        </n-space>

        <n-space vertical class="organize-weeks-space">
          <n-text>{{ t("organize.selectWeeks") }}:</n-text>
          <n-select
            v-model:value="selectedWeek"
            :options="weekOptions"
            :placeholder="t('organize.selectWeeksPlaceholder')"
            clearable
          />
        </n-space>
      </n-space>
    </n-card>

    <template v-if="selectedWeek != null">
      <n-card class="organize-card" :title="t('organize.weekDormPlaylist', { week: selectedWeek })">
        <DormPlaylistTable :dates="datesForWeek(selectedWeek)" :songs="songsStore.dormSongs" :timeSlots="settingsStore.timeSlots" />
        <n-space class="organize-week-actions">
          <n-button type="primary" :disabled="!copyMode" :loading="copyingWeek" @click="copyWeek(selectedWeek)">{{ t("organize.copyToSD") }}</n-button>
        </n-space>
      </n-card>
    </template>

    <n-empty v-else :description="t('organize.noWeeksSelected')" />

    <n-modal v-model:show="showConfirm" preset="dialog" :title="t('organize.confirmOverwrite')" positive-text="确认覆盖" negative-text="取消" @positive-click="doPendingCopy(true)">
      <n-ul>
        <n-li v-for="f in existingFiles" :key="f">{{ f }}</n-li>
      </n-ul>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useTheme } from "../composables/useTheme"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { organizeFiles, fetchSilentMP3, selectDir } from "../api/files"
import { useMessage } from "naive-ui"
import type { SelectOption } from "naive-ui"
import { dayjs } from "../utils/datetime"
import YearSelect from "../components/common/YearSelect.vue"
import DormPlaylistTable from "../components/dorm/DormPlaylistTable.vue"
import { buildExportEntries, hasAnySong } from "../utils/exportEntries"


const router = useRouter()
const { isDark } = useTheme()

function isTouchDevice() {
  return "ontouchstart" in window || navigator.maxTouchPoints > 0
}
function isMobileUA() {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
}
const shouldWarn = isMobileUA() || (isTouchDevice() && !matchMedia("(pointer: fine)").matches)
const warningSkipped = ref(false)
const showMobileWarning = computed(() => shouldWarn && !warningSkipped.value)

const logoSrc = computed(() => (isDark.value ? "/logo-white.png" : "/logo.png"))

const { t } = useI18n()
const songsStore = useSongsStore()
const settingsStore = useSettingsStore()
const message = useMessage()

const targetDir = ref("")
const targetDirName = ref("")
const targetDirHandle = ref<FileSystemDirectoryHandle | null>(null)
const copyMode = ref<"backend" | "frontend" | null>(null)
const selectedYear = ref(dayjs().year())
const selectedWeek = ref<number | null>(null)
const copyingWeek = ref<boolean>(false)
const showConfirm = ref(false)
const existingFiles = ref<string[]>([])
const pendingWeek = ref<number | null>(null)

function isoWeeksInYear(year: number): number {
  return dayjs(`${year}-12-28`).isoWeek()
}

const weekOptions = computed<SelectOption[]>(() => {
  const year = selectedYear.value
  const options: SelectOption[] = []
  const anchor = dayjs(`${year}-01-04`).startOf("isoWeek")
  const count = isoWeeksInYear(year)
  for (let week = 1; week <= count; week++) {
    const start = anchor.add(week - 1, "week")
    const last = start.endOf("isoWeek")
    options.push({
      label: `第${week}周 ${start.format("MM.DD")}-${last.format("MM.DD")}`,
      value: week,
    })
  }
  return options
})

function datesForWeek(week: number): string[] {
  const start = dayjs(`${selectedYear.value}-01-04`).startOf("isoWeek").add(week - 1, "week")
  const dates: string[] = []
  let cur = start
  for (let i = 0; i < 7; i++) {
    dates.push(cur.format("YYYY-MM-DD"))
    cur = cur.add(1, "day")
  }
  return dates
}

// 编号逻辑抽到 utils/exportEntries.ts，便于单测覆盖「空时段必须占位」这条规则。
function buildEntries(dates: string[]): { source: string; targetName: string }[] {
  return buildExportEntries(dates, songsStore.dormSongs, settingsStore.timeSlots)
}

async function browse() {
  // 1. 优先使用后端原生文件夹对话框（Windows 桌面环境）
  try {
    const path = await selectDir()
    if (path) {
      targetDir.value = path
      targetDirName.value = path
      targetDirHandle.value = null
      copyMode.value = "backend"
      return
    }
    // 用户取消
    copyMode.value = null
    return
  } catch (err) {
    console.log("Backend select-dir unavailable", err)
  }

  // 2. Fallback: File System Access API
  if ((window as any).showDirectoryPicker) {
    try {
      const handle = await (window as any).showDirectoryPicker()
      targetDirHandle.value = handle
      targetDirName.value = handle.name || ""
      targetDir.value = ""
      copyMode.value = "frontend"
    } catch (err: any) {
      if (err?.name !== "AbortError") {
        console.error(err)
        message.error(t("organize.selectFolderFailed"))
      }
      copyMode.value = null
    }
  } else {
    message.error("当前浏览器不支持文件夹选择")
    copyMode.value = null
  }
}

async function copyWeek(week: number) {
  if (!copyMode.value) {
    message.warning(t("organize.targetDirRequired"))
    return
  }
  pendingWeek.value = week
  await doPendingCopy(false)
}

async function doPendingCopy(confirm: boolean) {
  const week = pendingWeek.value
  if (week == null) return

  if (copyMode.value === "backend") {
    await doBackendCopy(week, confirm)
  } else if (copyMode.value === "frontend") {
    await doFrontendCopy(week, confirm)
  }
}

async function doBackendCopy(week: number, confirm: boolean) {
  const dates = datesForWeek(week)
  const entries = buildEntries(dates)
  if (!hasAnySong(entries)) {
    message.warning(t("organize.noSongsThisWeek"))
    pendingWeek.value = null
    return
  }

  copyingWeek.value = true
  try {
    const res = await organizeFiles({
      entries,
      targetDir: targetDir.value,
      mode: "copy",
      confirm,
    })
    if (res.confirmNeeded) {
      existingFiles.value = res.existingFiles || []
      showConfirm.value = true
      copyingWeek.value = false
      return
    }
    message.success(t("organize.success", { count: res.successful.length }))
    if (res.failed?.length > 0) {
      message.error(t("organize.failed", { count: res.failed.length }))
    }
  } catch (err: any) {
    message.error(err?.message || t("organize.organizeFailed"))
  } finally {
    copyingWeek.value = false
    pendingWeek.value = null
    showConfirm.value = false
  }
}

async function doFrontendCopy(week: number, confirm: boolean) {
  const dirHandle = targetDirHandle.value
  if (!dirHandle) return
  const dates = datesForWeek(week)
  const entries = buildEntries(dates)
  if (!hasAnySong(entries)) {
    message.warning(t("organize.noSongsThisWeek"))
    pendingWeek.value = null
    return
  }

  if (!confirm) {
    const existing: string[] = []
    for (const e of entries) {
      try {
        await dirHandle.getFileHandle(e.targetName)
        existing.push(e.targetName)
      } catch {
        // file does not exist
      }
    }
    if (existing.length > 0) {
      existingFiles.value = existing
      showConfirm.value = true
      copyingWeek.value = false
      return
    }
  }

  copyingWeek.value = true
  try {
    const perm = await (dirHandle as any).requestPermission({ mode: "readwrite" })
    if (perm !== "granted") {
      message.error("没有目录写入权限")
      copyingWeek.value = false
      pendingWeek.value = null
      return
    }

    let success = 0
    let failed = 0
    for (const entry of entries) {
      try {
        const blob = entry.source
          ? await fetchSourceBlob(entry.source)
          : await fetchSilentMP3(settingsStore.silentPlaceholderDuration || 30)
        await writeFile(dirHandle, entry.targetName, blob)
        success++
      } catch (err) {
        console.error("copy failed", entry, err)
        failed++
      }
    }

    message.success(t("organize.success", { count: success }))
    if (failed > 0) {
      message.error(t("organize.failed", { count: failed }))
    }
  } catch (err: any) {
    message.error(err?.message || t("organize.organizeFailed"))
  } finally {
    copyingWeek.value = false
    pendingWeek.value = null
    showConfirm.value = false
  }
}

async function fetchSourceBlob(filePath: string): Promise<Blob> {
  const res = await fetch(`/api/files/stream?file=${encodeURIComponent(filePath)}`)
  if (!res.ok) throw new Error("读取源文件失败")
  return await res.blob()
}

async function writeFile(dirHandle: FileSystemDirectoryHandle, name: string, blob: Blob) {
  const fileHandle = await dirHandle.getFileHandle(name, { create: true })
  const writable = await fileHandle.createWritable()
  await writable.write(blob)
  await writable.close()
}

onMounted(async () => {
  await Promise.all([songsStore.fetchSongs("dorm"), settingsStore.fetchSettings()])
})
</script>

<style scoped>
.organize-mobile-warning {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: var(--spacing-md);
  box-sizing: border-box;
}

.organize-mobile-logo {
  width: 120px;
  height: 120px;
}

.organize-mobile-title {
  margin-top: var(--spacing-lg);
  margin-bottom: 0;
}

.organize-mobile-text {
  max-width: 320px;
  text-align: center;
  margin-top: var(--spacing-sm);
}

.organize-mobile-actions {
  margin-top: var(--spacing-lg);
}

.organize-page {
  padding: var(--spacing-md);
  max-width: 1000px;
  margin: 0 auto;
}

.organize-card {
  margin-bottom: var(--spacing-md);
}

.organize-dir-input-field {
  width: 300px;
}

.organize-weeks-space {
  width: 100%;
}

.organize-week-actions {
  margin-top: var(--spacing-sm);
}

@media (max-width: 640px) {
  .organize-dir-input-field {
    width: 100%;
  }
}
</style>
