<template>
  <n-card v-if="songs.length > 0" style="margin-bottom:16px;" :title="title">
    <n-list>
      <n-list-item v-for="song in songs" :key="song.id">
        <n-space align="center" justify="space-between" style="width:100%;" wrap>
          <n-space align="center">
            <n-text strong>{{ song.title }}</n-text>
            <n-select
              :value="song.timeSlotId || null"
              :options="slotOptions"
              size="tiny"
              placeholder="时段"
              style="width:90px;"
              @update:value="(v: string | null) => $emit('update-slot', song.id, v || '')"
            />
          </n-space>
          <n-space align="center">
            <n-button v-if="song.filePath" size="tiny" @click="$emit('preview', song)">{{ t('common.listen') }}</n-button>
            <n-button size="tiny" type="error" @click="$emit('delete-song', song.id)">
              <template #icon>
                <Delete theme="outline" :size="14" :strokeWidth="3" />
              </template>
            </n-button>
          </n-space>
        </n-space>
      </n-list-item>
    </n-list>
  </n-card>
</template>

<script setup lang="ts">
import { useI18n } from "../../i18n"
import { Delete } from "@icon-park/vue-next"
import type { Song } from "../../api/types"
import type { SelectOption } from "naive-ui"

const { t } = useI18n()

defineProps<{
  title: string
  songs: Song[]
  slotName: (slotId: string | null) => string
  slotOptions: SelectOption[]
}>()

defineEmits<{
  (e: "preview", song: Song): void
  (e: "update-slot", songId: number, slotId: string): void
  (e: "delete-song", songId: number): void
}>()
</script>
