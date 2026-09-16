<template>
  <div style="padding:24px;max-width:860px;margin:0 auto;">
    <n-h2 style="margin:0 0 16px;color:var(--theme-color);">{{ t("guide.title") }}</n-h2>

    <n-card>
      <n-collapse :default-expanded-names="['import']" arrow-placement="left">
        <n-collapse-item :title="t('guide.import')" name="import">
          <n-ol>
            <n-li>{{ t("guide.importDesc1") }}</n-li>
            <n-li>{{ t("guide.importDesc2") }}</n-li>
            <n-li>{{ t("guide.importDesc3") }}</n-li>
            <n-li>{{ t("guide.importDesc4") }}</n-li>
          </n-ol>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.dorm')" name="dorm">
          <n-p>{{ t("guide.dormDesc1") }}</n-p>
          <n-p>{{ t("guide.dormDesc2") }}</n-p>
          <n-p>{{ t("guide.dormDesc3") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.broadcast')" name="broadcast">
          <n-p>{{ t("guide.broadcastDesc1") }}</n-p>
          <n-p>{{ t("guide.broadcastDesc2") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.export')" name="export">
          <n-p>{{ t("guide.exportDesc1") }}</n-p>
          <n-p>{{ t("guide.exportDesc2") }}</n-p>
          <n-p>{{ t("guide.exportDesc3") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.organize')" name="organize">
          <n-p>{{ t("guide.organizeDesc1") }}</n-p>
          <n-p>{{ t("guide.organizeDesc2") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.mobile')" name="mobile">
          <n-p>{{ t("guide.mobileDesc1") }}</n-p>
          <n-p>{{ t("guide.mobileDesc2") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.decrypt')" name="decrypt">
          <n-p>{{ t("guide.decryptDesc1") }}</n-p>
          <n-p>{{ t("guide.decryptDesc2") }}</n-p>
        </n-collapse-item>

        <n-collapse-item :title="t('guide.resetPassword')" name="resetPassword">
          <n-p>{{ t("guide.resetPasswordDesc1") }}</n-p>
          <n-ol>
            <n-li>{{ t("guide.resetPasswordDesc2") }}</n-li>
            <n-li>{{ t("guide.resetPasswordDesc3") }}</n-li>
          </n-ol>
        </n-collapse-item>
      </n-collapse>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "../i18n"

const { t } = useI18n()
</script>
