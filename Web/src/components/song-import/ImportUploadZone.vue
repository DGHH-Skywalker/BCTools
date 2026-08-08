<template>
  <n-card style="margin-bottom:16px;" :title="title">
    <n-upload
      v-model:file-list="fileList"
      :default-upload="false"
      :accept="accept"
      multiple
      @change="onChange"
    >
      <n-upload-dragger>
        <div style="padding:24px;text-align:center;">
          <n-h3>{{ t("songImport.dropFiles") }}</n-h3>
          <n-p depth="3">{{ formats }}</n-p>
        </div>
      </n-upload-dragger>
    </n-upload>
  </n-card>
</template>

<script setup lang="ts">
import { ref, nextTick } from "vue"
import { useI18n } from "../../i18n"

const { t } = useI18n()

defineProps<{
  title: string
  accept: string
  formats: string
}>()

const emit = defineEmits<{
  (e: "change", value: any): void
}>()

// Keep n-upload's internal file list bound so we can clear it after each
// selection. Without this, `multiple` + `default-upload=false` causes the
// component to accumulate previously imported files; the next import would
// re-submit A along with the newly selected B.
const fileList = ref<any[]>([])

function onChange(event: any) {
  emit("change", event)
  // Clear after the parent has processed the current batch so the next
  // selection only contains newly dropped/selected files.
  nextTick(() => {
    fileList.value = []
  })
}
</script>
