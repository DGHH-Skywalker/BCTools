<template>
  <div style="padding:24px;max-width:800px;margin:0 auto;">
    <PageTitle :title="t('settings.title')" />
    <n-card style="margin-bottom:16px;">
      <n-space vertical>
        <n-space align="center">
          <n-text>{{ t("settings.version") }}: {{ settingsStore.version }}</n-text>
          <n-button size="small" @click="checkUpdate">{{ t("settings.checkUpdate") }}</n-button>
          <n-button size="small" @click="showGuideModal = true">{{ t("settings.guide") }}</n-button>
        </n-space>
        <n-space align="center" v-if="config.languageSwitchEnabled">
          <n-text>{{ t("settings.language") }}:</n-text>
          <n-select v-model:value="localeVal" :options="[{label:'中文',value:'zh-CN'},{label:'English',value:'en'}]" style="width:120px" @update:value="switchLang" />
        </n-space>
      </n-space>
    </n-card>
    <n-card style="margin-bottom:16px;">
      <n-space vertical>
        <n-space align="center">
          <n-text>{{ t("settings.autoBackup") }}: </n-text>
          <n-switch v-model:value="settingsStore.autoBackupEnabled" @update:value="saveBackupSetting" />
        </n-space>
        <n-space align="center" v-if="settingsStore.autoBackupEnabled">
          <n-input v-model:value="settingsStore.autoBackupPath" placeholder="选择备份路径" style="width:300px" />
          <n-button @click="browsePath">{{ t("settings.browsePath") }}</n-button>
        </n-space>
        <n-button @click="backupNow">{{ t("settings.backupNow") }}</n-button>
      </n-space>
    </n-card>
    <n-modal
      v-model:show="showGuideModal"
      title="软件指南"
      preset="card"
      :style="{ width: '80%', maxWidth: '900px' }"
      :content-style="{ padding: 0 }"
    >
      <n-scrollbar style="max-height: 70vh; padding: 24px;">
        <n-p>这是软件指南的测试内容第 1 行。</n-p>
        <n-p>这是软件指南的测试内容第 2 行。</n-p>
        <n-p>这是软件指南的测试内容第 3 行。</n-p>
        <n-p>后续将在此补充完整的使用说明、常见问题与操作步骤。</n-p>
      </n-scrollbar>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useAppConfig } from "../composables/useAppConfig"
import { backup } from "../api/sync"
import { selectDir } from "../api/files"
import PageTitle from "../components/common/PageTitle.vue"
import { useMessage } from "naive-ui"

const { t, setLocale, currentLocale } = useI18n()
const settingsStore = useSettingsStore()
const { config } = useAppConfig()
const message = useMessage()
const showGuideModal = ref(false)
const localeVal = ref(currentLocale.value)

onMounted(() => {
  settingsStore.fetchSettings()
})

async function checkUpdate() {
  const result = await settingsStore.checkUpdate()
  if (result.hasUpdate) message.info(t("settings.newVersion", { version: result.latestVersion }))
  else message.success(t("settings.latestVersion"))
}

function switchLang(val: string) {
  setLocale(val as any)
  settingsStore.updateSettings({ locale: val } as any)
}

async function saveBackupSetting(val: boolean) {
  await settingsStore.updateSettings({ autoBackupEnabled: val } as any)
}

async function browsePath() {
  const path = await selectDir()
  if (path) {
    settingsStore.autoBackupPath = path
    await settingsStore.updateSettings({ autoBackupPath: path } as any)
  }
}

async function backupNow() {
  try {
    await backup()
    message.success(t("settings.backupSuccess"))
  } catch { message.error(t("settings.backupFailed", { reason: "未知" })) }
}
</script>
