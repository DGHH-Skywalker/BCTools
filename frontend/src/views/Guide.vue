<template>
  <div style="padding:24px;max-width:800px;margin:0 auto;">
    <n-h2>{{ t("guide.title") }}</n-h2>
    <n-collapse>
      <n-collapse-item title="导入歌曲" name="1">
        <n-ol>
          <n-li>{{ t("guide.importDesc1") }}</n-li>
          <n-li>{{ t("guide.importDesc2") }}</n-li>
          <n-li>{{ t("guide.importDesc3") }}</n-li>
          <n-li>{{ t("guide.importDesc4") }}</n-li>
        </n-ol>
      </n-collapse-item>
      <n-collapse-item title="分配歌曲" name="2">
        <n-p>{{ t("guide.assignDesc1") }}</n-p>
        <n-p>{{ t("guide.assignDesc2") }}</n-p>
      </n-collapse-item>
      <n-collapse-item title="导出歌单" name="3">
        <n-p>{{ t("guide.exportDesc1") }}</n-p>
        <n-p>{{ t("guide.exportDesc2") }}</n-p>
        <n-p>{{ t("guide.exportDesc3") }}</n-p>
      </n-collapse-item>
      <n-collapse-item title="文件整理" name="4">
        <n-p>{{ t("guide.organizeDesc1") }}</n-p>
        <n-p>{{ t("guide.organizeDesc2") }}</n-p>
      </n-collapse-item>
      <n-collapse-item title="备份与恢复" name="5">
        <n-p>{{ t("guide.backupDesc1") }}</n-p>
        <n-p>{{ t("guide.backupDesc2") }}</n-p>
      </n-collapse-item>
      <n-collapse-item title="密码重置" name="6">
        <n-p>{{ t("guide.resetPasswordDesc1") }}</n-p>
        <n-p>{{ t("guide.resetPasswordDesc2") }}</n-p>
        <n-p>{{ t("guide.resetPasswordDesc3") }}</n-p>
      </n-collapse-item>
    </n-collapse>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"
const { t } = useI18n()
</script>
