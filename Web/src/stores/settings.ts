import { defineStore } from "pinia"
import { ref } from "vue"
import type { Settings, TimeSlot } from "../api/types"
import * as settingsApi from "../api/misc"

export const useSettingsStore = defineStore("settings", () => {
  const timeSlots = ref<TimeSlot[]>([])
  const allowTemplateJS = ref(false)
  const silentPlaceholderDuration = ref(30)
  const version = ref("5.5.0.0")
  const downloadUrl = ref("")
  const adminPasswordHint = ref("")
  const locale = ref("zh-CN")
  const broadcastColumnMap = ref<Record<string, string>>({})
  const duplicateCheckDays = ref(30)
  const loading = ref(false)

  async function fetchSettings() {
    loading.value = true
    try {
      const s: Settings = await settingsApi.getSettings()
      timeSlots.value = s.timeSlots || []
      allowTemplateJS.value = s.allowTemplateJS
      silentPlaceholderDuration.value = s.silentPlaceholderDuration || 30
      version.value = s.version
      downloadUrl.value = s.downloadUrl
      adminPasswordHint.value = s.adminPasswordHint
      locale.value = s.locale || "zh-CN"
      broadcastColumnMap.value = s.broadcastColumnMap || {}
      duplicateCheckDays.value = s.duplicateCheckDays || 30
    } finally {
      loading.value = false
    }
  }

  async function updateSettings(data: Partial<Settings> & { confirmed?: boolean }) {
    const result: any = await settingsApi.updateSettings(data)
    if (result.timeSlots) timeSlots.value = result.timeSlots
    if (result.broadcastColumnMap) broadcastColumnMap.value = result.broadcastColumnMap
    if (result.duplicateCheckDays !== undefined) duplicateCheckDays.value = result.duplicateCheckDays
    if (result.confirmNeeded) return result
    return result
  }

  return {
    timeSlots, allowTemplateJS,
    silentPlaceholderDuration, version, downloadUrl, adminPasswordHint,
    locale, broadcastColumnMap, duplicateCheckDays, loading, fetchSettings, updateSettings,
  }
})
