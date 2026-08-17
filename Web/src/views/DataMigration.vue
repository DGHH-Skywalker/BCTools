<template>
  <PageContainer>
    <n-space align="center" style="margin-bottom: 16px">
      <n-button size="small" @click="router.push('/settings')">
        <template #icon><Left theme="outline" :size="14" :strokeWidth="3" /></template>
        {{ t("migration.back") }}
      </n-button>
      <n-h2 style="margin: 0">{{ t("migration.title") }}</n-h2>
    </n-space>

    <n-p depth="3" style="max-width: 720px; margin-top: 0">
      {{ t("migration.subtitle") }}
    </n-p>

    <n-grid cols="1 m:2" :x-gap="16" :y-gap="16" responsive="screen">
      <!-- 左：小清单（不带歌） -->
      <n-grid-item>
        <n-card :title="t('migration.jsonMode')" size="small" class="migration-card">
          <n-p depth="3" style="margin-top: 0">
            {{ t("migration.jsonModeDesc") }}
          </n-p>
          <n-button
            type="primary"
            :loading="jsonExporting"
            :disabled="jsonExporting"
            @click="handleExportJson"
          >
            {{ t("migration.exportJsonButton") }}
          </n-button>
          <n-progress
            v-if="jsonExporting"
            :show-indicator="false"
            :percentage="100"
            :indicator-placement="'inside'"
            processing
            style="margin-top: 12px"
          />
          <n-text v-if="jsonExporting" depth="3" style="font-size: 12px">
            {{ t("migration.progressJson") }}
          </n-text>
        </n-card>
      </n-grid-item>

      <!-- 右：完整搬家（推荐） -->
      <n-grid-item>
        <n-card :title="t('migration.filesMode')" size="small" class="migration-card">
          <n-p depth="3" style="margin-top: 0">
            {{ t("migration.filesModeDesc") }}
          </n-p>

          <n-space align="center" style="margin-bottom: 12px" wrap>
            <n-text>{{ t("migration.selectTargetDir") }}:</n-text>
            <n-input
              v-model:value="targetDir"
              :placeholder="t('migration.noTargetDir')"
              class="migration-dir-input"
            />
            <n-button :loading="selectingDir" @click="handleSelectDir">
              {{ t("organize.browse") }}
            </n-button>
          </n-space>

          <n-text v-if="previewHint" depth="3" style="display: block; margin-bottom: 12px">
            {{ previewHint }}
          </n-text>

          <n-button
            type="primary"
            :disabled="!targetDir || filesExporting"
            :loading="filesExporting"
            @click="handleExportFiles"
          >
            {{ t("migration.exportFilesButton") }}
          </n-button>
          <n-progress
            v-if="filesExporting"
            :show-indicator="true"
            :percentage="filesProgressPct"
            :indicator-placement="'inside'"
            :status="filesProgressStatus"
            style="margin-top: 12px"
          />
          <n-text v-if="filesExporting" depth="3" style="font-size: 12px">
            {{ t("migration.progressFiles", { step: t(filesProgressStep) }) }}
          </n-text>
        </n-card>
      </n-grid-item>
    </n-grid>

    <!-- 冲突 modal：目标子目录已存在 -->
    <n-modal
      v-model:show="showConflict"
      preset="dialog"
      :title="t('migration.conflictTitle')"
      :content="conflictContent"
      :positive-text="t('migration.conflictOverwrite')"
      :negative-text="t('migration.conflictChangeDir')"
      @positive-click="onConflictOverwrite"
      @negative-click="onConflictChangeDir"
    />

    <!-- 装回：与导出分开一节 -->
    <n-divider style="margin: 32px 0 16px" />

    <n-h3 style="margin: 0 0 4px">{{ t("migration.importSection") }}</n-h3>
    <n-p depth="3" style="margin-top: 0; max-width: 720px">
      {{ t("migration.importDescription") }}
    </n-p>

    <n-card size="small" style="max-width: 720px">
      <n-tabs default-value="folder" type="segment" size="small">
        <!-- Tab 1: 从文件夹装回 -->
        <n-tab-pane name="folder" :tab="t('migration.importFromFolder')">
          <n-space align="center" style="margin-top: 12px; margin-bottom: 8px" wrap>
            <n-text>{{ t("migration.importSelectDir") }}:</n-text>
            <n-input
              v-model:value="importDir"
              :placeholder="t('migration.importNoDir')"
              class="migration-dir-input"
            />
            <n-button :loading="selectingDir" @click="handleSelectImportDir">
              {{ t("organize.browse") }}
            </n-button>
          </n-space>
        </n-tab-pane>

        <!-- Tab 2: 从 JSON 文件装回 -->
        <n-tab-pane name="file" :tab="t('migration.importFromFile')">
          <n-space align="center" style="margin-top: 12px; margin-bottom: 8px" wrap>
            <n-text>{{ t("migration.importSelectFile") }}:</n-text>
            <n-input
              v-model:value="importFileName"
              :placeholder="t('migration.importNoFile')"
              class="migration-dir-input"
              disabled
            />
            <n-button @click="triggerFilePicker">
              {{ t("organize.browse") }}
            </n-button>
            <input
              ref="fileInputEl"
              type="file"
              accept=".json,application/json"
              style="display: none"
              @change="onFileSelected"
            />
          </n-space>
        </n-tab-pane>
      </n-tabs>

      <n-radio-group v-model:value="importMode" style="margin: 12px 0 8px">
        <n-space>
          <n-radio value="merge">{{ t("migration.importModeMerge") }}</n-radio>
          <n-radio value="replace">{{ t("migration.importModeReplace") }}</n-radio>
        </n-space>
      </n-radio-group>
      <n-text depth="3" style="display: block; font-size: 12px; margin-bottom: 12px">
        {{ importMode === "merge" ? t("migration.importModeMergeHint") : t("migration.importModeReplaceHint") }}
      </n-text>

      <n-button
        type="primary"
        :disabled="(!importDir && !importFilePayload) || importing"
        :loading="importing"
        @click="handleImport"
      >
        {{ t("migration.importConfirm") }}
      </n-button>
      <n-progress
        v-if="importing"
        :show-indicator="true"
        :percentage="importProgressPct"
        :indicator-placement="'inside'"
        :status="importProgressStatus"
        style="margin-top: 12px"
      />
      <n-text v-if="importing" depth="3" style="font-size: 12px">
        {{ t(importingFromFile ? "migration.progressImportFile" : "migration.progressImport", { step: t(importProgressStep) }) }}
      </n-text>
    </n-card>
  </PageContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useMessage } from "naive-ui"
import { Left } from "@icon-park/vue-next"
import {
  exportMigrationJson,
  exportMigrationFiles,
  previewMigration,
  importMigration,
} from "../api/migration"
import { selectDir } from "../api/files"
import PageContainer from "../components/ui/PageContainer.vue"

const { t } = useI18n()
const router = useRouter()
const message = useMessage()

// ===== 导出 =====
const targetDir = ref("")
const selectingDir = ref(false)
const jsonExporting = ref(false)
const filesExporting = ref(false)
const previewHint = ref("")
const filesProgressPct = ref(0)
const filesProgressStep = ref("migration.progressStepCopying")
const filesProgressStatus = ref<"default" | "success" | "error" | "warning">("default")

// ===== 冲突 modal =====
const showConflict = ref(false)
const conflictDir = ref("")
const conflictCount = ref(0)

const conflictContent = computed(() => {
  const dirName = conflictDir.value.split(/[\\/]/).pop() || conflictDir.value
  return t("migration.conflictContent", { dir: dirName, count: conflictCount.value })
})

// ===== 导入 =====
const importDir = ref("")
const importFileName = ref("")
const importFilePayload = ref<any>(null)
const importing = ref(false)
const importingFromFile = ref(false)
const importMode = ref<"merge" | "replace">("merge")
const importProgressPct = ref(0)
const importProgressStep = ref("migration.progressStepCopying")
const importProgressStatus = ref<"default" | "success" | "error" | "warning">("default")
const fileInputEl = ref<HTMLInputElement | null>(null)

function triggerFilePicker() {
  fileInputEl.value?.click()
}

async function onFileSelected(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  if (!f) return
  try {
    const text = await f.text()
    const parsed = JSON.parse(text)
    importFilePayload.value = parsed
    importFileName.value = f.name
  } catch (err: any) {
    importFilePayload.value = null
    importFileName.value = ""
    message.error(t("migration.importInvalidBackup", { msg: err?.message || "JSON 解析失败" }))
  } finally {
    input.value = ""
  }
}

onMounted(() => {})

async function pickFolder(): Promise<string> {
  try {
    const path = await selectDir()
    if (path) return path
  } catch (err) {
    console.log("Backend select-dir unavailable", err)
  }
  if ((window as any).showDirectoryPicker) {
    try {
      const handle = await (window as any).showDirectoryPicker()
      message.warning(handle?.name || "")
    } catch (err: any) {
      if (err?.name !== "AbortError") {
        message.error(t("migration.selectDirFailed"))
      }
    }
  } else {
    message.error(t("migration.noBrowserFolderSupport"))
  }
  return ""
}

async function handleSelectDir() {
  selectingDir.value = true
  try {
    const path = await pickFolder()
    if (path) {
      targetDir.value = path
      await refreshPreview()
    }
  } finally {
    selectingDir.value = false
  }
}

async function handleSelectImportDir() {
  selectingDir.value = true
  try {
    const path = await pickFolder()
    if (path) {
      importDir.value = path
    }
  } finally {
    selectingDir.value = false
  }
}

async function refreshPreview() {
  if (!targetDir.value) {
    previewHint.value = ""
    return
  }
  try {
    const preview = await previewMigration(targetDir.value)
    const dirName = preview.backupDir.split(/[\\/]/).pop() || preview.backupDir
    previewHint.value = t("migration.targetDirChanged", { path: dirName })
  } catch (err: any) {
    previewHint.value = ""
    message.error(
      t("migration.previewFailed", { msg: err?.response?.data?.error?.message || err?.message || "" }),
    )
  }
}

function startFilesProgress() {
  filesProgressPct.value = 0
  filesProgressStep.value = "migration.progressStepCopying"
  filesProgressStatus.value = "default"
  const tick = setInterval(() => {
    if (!filesExporting.value) { clearInterval(tick); return }
    if (filesProgressPct.value < 60) filesProgressPct.value += 6
    else if (filesProgressPct.value < 90) filesProgressPct.value += 2
  }, 80)
}

function finishFilesProgress(success: boolean) {
  filesProgressPct.value = 100
  filesProgressStep.value = "migration.progressStepWriting"
  filesProgressStatus.value = success ? "success" : "error"
  setTimeout(() => {
    filesExporting.value = false
    filesProgressPct.value = 0
  }, 800)
}

async function handleExportJson() {
  jsonExporting.value = true
  try {
    await exportMigrationJson()
    message.success(t("migration.jsonSuccess"))
  } catch (err: any) {
    message.error(err?.message || t("migration.filesFailed", { msg: "" }))
  } finally {
    jsonExporting.value = false
  }
}

async function handleExportFiles() {
  if (!targetDir.value) {
    message.warning(t("organize.targetDirRequired"))
    return
  }
  await doExportFiles(false)
}

async function doExportFiles(confirm: boolean) {
  filesExporting.value = true
  startFilesProgress()
  try {
    const res = await exportMigrationFiles(targetDir.value, confirm)
    if (res.confirmNeeded) {
      conflictDir.value = res.backupDir || ""
      conflictCount.value = (res.existingFiles || []).length
      showConflict.value = true
      filesExporting.value = false
      filesProgressPct.value = 0
      return
    }
    finishFilesProgress(true)
    const written = res.writtenFiles?.length || 0
    message.success(t("migration.filesSuccess", { count: written, path: res.backupDir || "" }))
    if (res.skippedFiles && res.skippedFiles.length > 0) {
      message.warning(t("migration.filesSkipped", { count: res.skippedFiles.length }))
    }
    await refreshPreview()
  } catch (err: any) {
    finishFilesProgress(false)
    const msg = err?.response?.data?.error?.message || err?.message || ""
    message.error(t("migration.filesFailed", { msg }))
  }
}

async function onConflictOverwrite() {
  await doExportFiles(true)
}

async function onConflictChangeDir() {
  await handleSelectDir()
}

// ===== 导入 =====
function startImportProgress() {
  importProgressPct.value = 0
  importProgressStep.value = "migration.progressStepCopying"
  importProgressStatus.value = "default"
  const tick = setInterval(() => {
    if (!importing.value) { clearInterval(tick); return }
    if (importProgressPct.value < 60) importProgressPct.value += 6
    else if (importProgressPct.value < 90) importProgressPct.value += 2
  }, 80)
}

function finishImportProgress(success: boolean) {
  importProgressPct.value = 100
  importProgressStep.value = "migration.progressStepWriting"
  importProgressStatus.value = success ? "success" : "error"
  setTimeout(() => {
    importing.value = false
    importProgressPct.value = 0
  }, 800)
}

async function handleImport() {
  const isFile = !!importFilePayload.value
  if (isFile) {
    if (!importFilePayload.value) {
      message.warning(t("migration.importNoFile"))
      return
    }
  } else if (!importDir.value) {
    message.warning(t("migration.importNoDir"))
    return
  }
  importing.value = true
  importingFromFile.value = isFile
  startImportProgress()
  try {
    const res = await importMigration({
      ...(isFile ? { jsonData: importFilePayload.value } : { backupDir: importDir.value }),
      mode: importMode.value,
    })
    finishImportProgress(true)
    message.success(t("migration.importSuccess", {
      dorm: res.insertedDorm,
      broadcast: res.insertedBroadcast,
      files: res.filesCopied,
      slots: res.timeSlotsMerged,
    }))
    if (res.filesMissing && res.filesMissing.length > 0) {
      message.warning(t("migration.importMissing", { count: res.filesMissing.length }))
    }
  } catch (err: any) {
    finishImportProgress(false)
    const msg = err?.response?.data?.error?.message || err?.message || ""
    message.error(t("migration.importInvalidBackup", { msg }))
  }
}
</script>

<style scoped>
.migration-card {
  height: 100%;
}

.migration-dir-input {
  width: 320px;
  max-width: 100%;
}

@media (max-width: 640px) {
  .migration-dir-input {
    width: 100%;
  }
}
</style>
