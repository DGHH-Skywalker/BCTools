<template>
  <div class="song-row">
    <span class="drag-handle" :title="t('dorm.dragSort')">
      <svg viewBox="0 0 10 16" width="10" height="16" aria-hidden="true">
        <circle cx="2.5" cy="3" r="1.3" fill="currentColor" />
        <circle cx="2.5" cy="8" r="1.3" fill="currentColor" />
        <circle cx="2.5" cy="13" r="1.3" fill="currentColor" />
        <circle cx="7" cy="3" r="1.3" fill="currentColor" />
        <circle cx="7" cy="8" r="1.3" fill="currentColor" />
        <circle cx="7" cy="13" r="1.3" fill="currentColor" />
      </svg>
    </span>
    <div class="song-info">
      <span v-if="props.showDate" class="song-date">{{ props.song.date }}</span>
      <n-input
        class="no-drag"
        :value="props.song.title"
        size="small"
        :placeholder="t('broadcast.songTitle')"
        @update:value="(v: string) => emit('update-title', props.song.id, v)"
        @blur="emit('save-title', props.song.id)"
      />
      <div v-if="props.warnings.length" class="duplicate-warning">
        <div v-for="w in props.warnings" :key="w.id">
          {{ t("dorm.duplicateWarning", { date: w.date, title: w.title }) }}
        </div>
      </div>
    </div>

    <n-popover
      class="no-drag"
      trigger="click"
      placement="bottom"
      :show-arrow="false"
      :show="props.open"
      @clickoutside="emit('toggle-popover', null)"
    >
      <template #trigger>
        <n-button
          class="no-drag"
          size="tiny"
          :title="t('dorm.assignSlot')"
          @click="emit('toggle-popover', props.open ? null : props.song.id)"
        >
          {{ slotLabel }}
        </n-button>
      </template>
      <div class="slot-option-list">
        <div
          v-for="opt in props.slotOptions"
          :key="opt.value"
          class="slot-option"
          :class="{ active: opt.value === (props.song.timeSlotId ?? '__unassigned__') }"
          @click="emit('assign-slot', props.song.id, opt.value)"
        >{{ opt.label }}</div>
      </div>
    </n-popover>

    <AudioPlayButton
      v-if="props.song.filePath"
      class="no-drag"
      size="tiny"
      :file-path="props.song.filePath"
      :title="props.song.title"
    />
    <n-button class="no-drag" size="tiny" type="error" @click="emit('remove', props.song.id)">
      <template #icon>
        <Delete theme="outline" :size="14" :strokeWidth="3" />
      </template>
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "../../i18n"
import AudioPlayButton from "../common/AudioPlayButton.vue"
import { Delete } from "@icon-park/vue-next"
import type { Song } from "../../api/types"

interface SlotOption {
  label: string
  value: string
}

const props = defineProps<{
  song: Song
  slotOptions: SlotOption[]
  open: boolean
  warnings: Array<{ id: number; date: string; title: string }>
  showDate?: boolean
}>()

const emit = defineEmits<{
  (e: "update-title", id: number, title: string): void
  (e: "save-title", id: number): void
  (e: "remove", id: number): void
  (e: "assign-slot", id: number, slotId: string): void
  (e: "toggle-popover", id: number | null): void
}>()

const { t } = useI18n()

// 标签直接取自 slotOptions（按歌曲所在日期计算），因此时段被拖动或改名后
// 按钮文字会随之更新，不会停留在旧时段。
const slotLabel = computed(() => {
  const key = props.song.timeSlotId ?? "__unassigned__"
  return props.slotOptions.find((o) => o.value === key)?.label ?? t("dorm.unassigned")
})
</script>

<style scoped>
.song-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.drag-handle {
  cursor: grab;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}

.song-info {
  flex: 1;
  min-width: 0;
}

.song-date {
  display: block;
  font-size: 11px;
  color: var(--color-text-muted);
  line-height: 1.4;
}

.duplicate-warning {
  color: var(--color-warning);
  font-size: 12px;
  line-height: 1.4;
}

.slot-option-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 80px;
}

.slot-option {
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}

.slot-option:hover {
  background-color: var(--color-bg-light);
}

.slot-option.active {
  background-color: var(--color-primary-light);
  color: var(--color-primary);
}
</style>
