<template>
  <n-space vertical size="large" class="preferences-panel">
        <n-space align="center" wrap class="drawer-row">
          <n-text strong>{{ t("export.mode") }}</n-text>
          <n-switch v-model:value="exportStore.simpleMode" :rail-style="switchRailStyle">
            <template #checked>{{ t("export.simpleMode") }}</template>
            <template #unchecked>{{ t("export.posterMode") }}</template>
          </n-switch>
        </n-space>

        <n-space v-if="!exportStore.simpleMode" align="center" wrap class="drawer-row">
          <n-upload :default-upload="false" :show-file-list="false" accept="image/*" @change="(o) => onBackgroundUploadChange(o, 'dorm')">
            <div
              class="bg-preview-card"
              :style="{ backgroundImage: exportStore.backgroundImageDorm ? `url(${exportStore.backgroundImageDorm})` : 'none' }"
            >
              <div v-if="!exportStore.backgroundImageDorm" class="bg-preview-placeholder">{{ t("export.dorm") }}{{ t("export.backgroundImage") }}</div>
              <div v-else class="bg-preview-overlay">{{ t("common.replace") }}</div>
            </div>
          </n-upload>

          <n-upload :default-upload="false" :show-file-list="false" accept="image/*" @change="(o) => onBackgroundUploadChange(o, 'broadcast')">
            <div
              class="bg-preview-card"
              :style="{ backgroundImage: exportStore.backgroundImageBroadcast ? `url(${exportStore.backgroundImageBroadcast})` : 'none' }"
            >
              <div v-if="!exportStore.backgroundImageBroadcast" class="bg-preview-placeholder">{{ t("export.broadcast") }}{{ t("export.backgroundImage") }}</div>
              <div v-else class="bg-preview-overlay">{{ t("common.replace") }}</div>
            </div>
          </n-upload>
        </n-space>

        <template v-if="!exportStore.simpleMode">
          <n-space vertical class="drawer-section">
            <n-text strong>{{ t("export.vacationBadge") }}</n-text>
            <n-radio-group v-model:value="exportStore.vacationBadge" vertical>
              <n-radio value="none">{{ t("export.badgeNone") }}</n-radio>
              <n-radio value="summer">{{ t("export.badgeSummer") }}</n-radio>
              <n-radio value="winter">{{ t("export.badgeWinter") }}</n-radio>
            </n-radio-group>
          </n-space>

          <n-space align="start" class="drawer-row">
            <n-space vertical class="drawer-column">
              <n-text strong>{{ t("export.posterQuoteDorm") }}</n-text>
              <n-input
                v-model:value="exportStore.posterQuoteDorm"
                type="textarea"
                :placeholder="t('export.posterQuoteDormPlaceholder')"
                :autosize="{ minRows: 2, maxRows: 4 }"
                class="drawer-textarea"
              />
            </n-space>
            <n-space vertical class="drawer-column">
              <n-text strong>{{ t("export.posterQuoteBroadcast") }}</n-text>
              <n-input
                v-model:value="exportStore.posterQuoteBroadcast"
                type="textarea"
                :placeholder="t('export.posterQuoteBroadcastPlaceholder')"
                :autosize="{ minRows: 2, maxRows: 4 }"
                class="drawer-textarea"
              />
            </n-space>
          </n-space>

          <n-space align="start" class="drawer-row">
            <n-space vertical class="drawer-column">
              <n-text strong>{{ t("export.posterBlurDorm") }}: {{ blurLabel('dorm') }}</n-text>
              <div class="blur-preview" :style="blurPreviewStyle('dorm')">
                <span v-if="!exportStore.backgroundImageDorm" class="blur-placeholder">{{ t("export.blurPlaceholder") }}</span>
              </div>
              <n-slider v-model:value="exportStore.posterBlurDorm" :min="0" :max="20" :step="0.5" />
            </n-space>
            <n-space vertical class="drawer-column">
              <n-text strong>{{ t("export.posterBlurBroadcast") }}: {{ blurLabel('broadcast') }}</n-text>
              <div class="blur-preview" :style="blurPreviewStyle('broadcast')">
                <span v-if="!exportStore.backgroundImageBroadcast" class="blur-placeholder">{{ t("export.blurPlaceholder") }}</span>
              </div>
              <n-slider v-model:value="exportStore.posterBlurBroadcast" :min="0" :max="20" :step="0.5" />
            </n-space>
          </n-space>
        </template>
  </n-space>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "../../i18n"
import { useExportStore } from "../../stores/export"
import { useAppConfig } from "../../composables/useAppConfig"
import { useMessage } from "naive-ui"
import type { UploadFileInfo } from "naive-ui"
import type { SongType } from "../../api/types"
import { COLORS } from "../../constants/colors"

const { t } = useI18n()
const exportStore = useExportStore()
const { config } = useAppConfig()
const message = useMessage()
const themeColor = computed(() => config.value.themeColor)

function blurLabel(type: SongType) {
  const blur = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
  return `${blur.toFixed(1)}px`
}

function blurPreviewStyle(type: SongType) {
  const bg = exportStore.backgroundImageFor(type)
  const blur = type === "dorm" ? exportStore.posterBlurDorm : exportStore.posterBlurBroadcast
  const style: Record<string, string> = {
    backgroundImage: bg ? `url(${bg})` : "none",
    backgroundSize: "cover",
    backgroundPosition: "center",
    filter: `blur(${blur}px) brightness(0.95)`,
  }
  if (!bg) {
    style.background = themeColor.value
  }
  return style
}

function switchRailStyle({ checked }: { checked: boolean }) {
  return {
    background: checked ? COLORS.text : themeColor.value,
  }
}

function onBackgroundUploadChange({ fileList }: { fileList: UploadFileInfo[] }, type: "dorm" | "broadcast") {
  const file = fileList[0]?.file as File
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    exportStore.setBackgroundImage(type, reader.result as string)
  }
  reader.onerror = () => {
    message.error(t("export.backgroundImageError"))
  }
  reader.readAsDataURL(file)
}
</script>

<style>
.n-upload-file-list {
  display: none !important;
}
</style>

<style scoped>
.preferences-panel {
  width: 100%;
  max-width: 900px;
}

.drawer-row {
  width: 100%;
}

.drawer-section {
  width: 100%;
}

.drawer-column {
  flex: 1;
}

.drawer-textarea {
  width: 100%;
}

.blur-preview {
  width: 100%;
  aspect-ratio: 9 / 16;
  max-width: 180px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  overflow: hidden;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  justify-content: center;
}

.blur-placeholder {
  color: var(--color-white);
  font-size: 14px;
  opacity: 0.9;
}

.bg-preview-card {
  width: 80px;
  height: 56px;
  border-radius: var(--radius-sm);
  border: 1px dashed var(--theme-color, v-bind('COLORS.primary'));
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  cursor: pointer;
  position: relative;
}

.bg-preview-placeholder {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-align: center;
  padding: 4px;
  line-height: 1.2;
}

.bg-preview-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-overlay);
  color: var(--color-white);
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.15s;
}

.bg-preview-card:hover .bg-preview-overlay {
  opacity: 1;
}
</style>
