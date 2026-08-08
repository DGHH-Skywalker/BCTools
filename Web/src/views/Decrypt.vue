<template>
  <div style="padding:24px;max-width:800px;margin:0 auto;">
    <n-h2 style="margin-bottom:16px;">{{ t("decrypt.title") }}</n-h2>
    <n-card :bordered="true">
      <n-space vertical :size="12">
        <n-p>{{ t("decrypt.description") }}</n-p>
        <n-p style="color:var(--n-text-color-3);margin:0;">{{ t("decrypt.note") }}</n-p>
        <n-button type="primary" @click="openTool">
          <template #icon>
            <Key theme="outline" :size="18" :strokeWidth="3" />
          </template>
          {{ t("decrypt.openTool") }}
        </n-button>
      </n-space>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
import { Key } from "@icon-park/vue-next"

const { t } = useI18n()

// um-react 由后端 /um-react/* 托管，在新标签页打开；解密均在浏览器本地完成
function openTool() {
  window.open("/um-react/", "_blank")
}
</script>
