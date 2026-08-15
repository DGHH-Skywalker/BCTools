<template>
  <div>
    <n-p v-if="files.length > 0">{{ t("songImport.progress", { done: doneCount, total: files.length }) }}</n-p>

    <div v-if="files.length > 0" class="import-grid">
      <n-card v-for="file in files" :key="file.id" :class="['conv-card', file.status]" size="small">
        <n-space vertical class="conv-card-body">
          <n-space align="center" justify="space-between" class="conv-card-header">
            <n-space align="center" class="conv-card-info">
              <n-tag :type="statusTagType(file.status)" size="small" class="conv-card-tag">{{ file.fileType }}</n-tag>
              <n-ellipsis class="conv-card-name" :tooltip="{ disabled: false }">
                <span>{{ file.fileName }}</span>
              </n-ellipsis>
            </n-space>
            <n-button v-if="file.status==='error'" size="small" class="conv-card-action" @click="$emit('retry', file)">{{ t("songImport.retry") }}</n-button>
            <AudioPlayButton
              v-if="file.status==='done' && file.tempFileName"
              class="conv-card-action"
              size="small"
              :file-path="file.tempFileName"
              :title="file.songTitle"
            />
            <n-button v-if="file.status==='done'" size="small" type="error" class="conv-card-action" @click="$emit('delete', file)">{{ t("common.delete") }}</n-button>
          </n-space>

          <n-spin v-if="file.status==='converting'" size="small">
            <template #description>处理中...</template>
          </n-spin>

          <n-space v-if="file.status==='done'" vertical class="conv-card-meta">
            <n-input
              :value="file.songTitle"
              size="small"
              class="conv-card-title-input"
              :placeholder="t('songImport.editTitle')"
              @update:value="(v: string) => $emit('update-title', file, v)"
              @blur="$emit('save-metadata', file)"
            />
            <div
              v-if="file.duplicateWarnings && file.duplicateWarnings.length > 0"
              class="duplicate-warning"
            >
              <div v-for="s in file.duplicateWarnings" :key="s.id">
                {{ t("dorm.duplicateWarning", { date: s.date, title: s.title }) }}
              </div>
            </div>
            <n-select
              v-if="file.songId"
              size="small"
              :value="file.timeSlotId"
              :placeholder="t('songImport.slotPlaceholder')"
              :options="slotOptions"
              :render-label="renderSlotLabel"
              class="conv-card-slot"
              @update:value="(v: string | null) => $emit('assign-slot', file, v as string)"
            />
            <n-text v-if="file.timeSlotId" type="success" depth="3">{{ t("songImport.assigned") }}</n-text>
          </n-space>

          <n-text v-if="file.status==='error'" type="error" depth="3">{{ file.error }}</n-text>
        </n-space>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../../i18n"
import AudioPlayButton from "../common/AudioPlayButton.vue"
import type { ProcessingFile } from "../../composables/useFileProcessor"
import type { SelectOption } from "naive-ui"

const { t } = useI18n()

defineProps<{
  files: ProcessingFile[]
  doneCount: number
  slotOptions: SelectOption[]
  renderSlotLabel: (option: SelectOption) => string
  statusTagType: (status: ProcessingFile["status"]) => "success" | "error" | "warning"
}>()

defineEmits<{
  (e: "retry", file: ProcessingFile): void
  (e: "update-title", file: ProcessingFile, value: string): void
  (e: "save-metadata", file: ProcessingFile): void
  (e: "assign-slot", file: ProcessingFile, slotId: string): void
  (e: "delete", file: ProcessingFile): void
}>()
</script>

<style scoped>
.conv-card {
  margin-bottom: 4px;
}

.conv-card.error {
  border-color: var(--color-error);
}

.conv-card-body,
.conv-card-meta {
  width: 100%;
}

.conv-card-header {
  width: 100%;
}

.conv-card-info {
  min-width: 0;
  flex: 1;
}

.conv-card-tag,
.conv-card-action {
  flex-shrink: 0;
}

.conv-card-name {
  min-width: 0;
  flex: 1;
}

.conv-card-title-input,
.conv-card-slot {
  width: 100%;
}

.duplicate-warning {
  color: var(--color-warning);
  font-size: 12px;
  line-height: 1.4;
}

.import-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-top: 12px;
}

@media (max-width: 1024px) {
  .import-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .import-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 480px) {
  .import-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
