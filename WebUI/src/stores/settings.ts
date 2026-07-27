import { defineStore } from "pinia"
import { ref } from "vue"
import type { Settings, TimeSlot } from "../api/types"
import * as settingsApi from "../api/settings"
import * as updateApi from "../api/update"
import type { UpdateCheckResult } from "../api/types"

export const useSettingsStore = defineStore("settings", () => {
  const timeSlots = ref<TimeSlot[]>([])
  const allowTemplateJS = ref(false)
  const autoBackupPath = ref("")
  const autoBackupEnabled = ref(false)
  const silentPlaceholderDuration = ref(30)
  const version = ref("5.5.0.0")
  const downloadUrl = ref("")
  const adminPasswordHint = ref("")
  const locale = ref("zh-CN")
  const loading = ref(false)

  async function fetchSettings() {
    loading.value = true
    try {
      const s: Settings = await settingsApi.getSettings()
      timeSlots.value = s.timeSlots || []
      allowTemplateJS.value = s.allowTemplateJS
      autoBackupPath.value = s.autoBackupPath
      autoBackupEnabled.value = s.autoBackupEnabled
      silentPlaceholderDuration.value = s.silentPlaceholderDuration || 30
      version.value = s.version
      downloadUrl.value = s.downloadUrl
      adminPasswordHint.value = s.adminPasswordHint
      locale.value = s.locale || "zh-CN"
    } finally {
      loading.value = false
    }
  }

  async function updateSettings(data: Partial<Settings> & { confirmed?: boolean }) {
    const result: any = await settingsApi.updateSettings(data)
    if (result.timeSlots) timeSlots.value = result.timeSlots
    if (result.confirmNeeded) return result
    return result
  }

  async function checkUpdate(): Promise<UpdateCheckResult> {
    return await updateApi.checkUpdate()
  }

  return {
    timeSlots, allowTemplateJS, autoBackupPath, autoBackupEnabled,
    silentPlaceholderDuration, version, downloadUrl, adminPasswordHint,
    locale, loading, fetchSettings, updateSettings, checkUpdate,
  }
})
