<template>
  <div style="padding:24px;max-width:800px;margin:0 auto;">
    <n-h2>{{ t("settings.title") }}</n-h2>
    <n-card style="margin-bottom:16px;">
      <n-space vertical>
        <n-space align="center">
          <n-text>{{ t("settings.version") }}: {{ settingsStore.version }}</n-text>
          <n-button size="small" @click="checkUpdate">{{ t("settings.checkUpdate") }}</n-button>
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
    <n-card style="margin-bottom:16px;">
      <n-button @click="showPasswordModal = true">{{ t("settings.advanced") }}</n-button>
    </n-card>
    <n-modal v-model:show="showPasswordModal" :title="t('settings.advanced')" preset="card" style="width:400px;">
      <n-input v-model:value="password" type="password" :placeholder="t('settings.advancedPassword')" @keyup.enter="verifyPwd" />
      <n-p v-if="pwdError">{{ t("settings.passwordHint", { hint: pwdHint }) }}</n-p>
      <template #footer>
        <n-button @click="verifyPwd" type="primary">{{ t("common.confirm") }}</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import { useAppConfig } from "../composables/useAppConfig"
import { verifyPassword } from "../api/auth"
import { backup } from "../api/sync"
import { selectDir } from "../api/files"
import { useMessage } from "naive-ui"

const { t, setLocale, currentLocale } = useI18n()
const settingsStore = useSettingsStore()
const { config } = useAppConfig()
const router = useRouter()
const message = useMessage()
const showPasswordModal = ref(false)
const password = ref("")
const pwdError = ref(false)
const pwdHint = ref("")
const localeVal = ref(currentLocale.value)

onMounted(() => { settingsStore.fetchSettings() })

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

async function verifyPwd() {
  const result = await verifyPassword(password.value)
  if (result.success) {
    sessionStorage.setItem("advanced_authenticated", "1")
    showPasswordModal.value = false
    router.push("/advanced")
  } else {
    pwdError.value = true
    pwdHint.value = result.hint || ""
  }
}
</script>
