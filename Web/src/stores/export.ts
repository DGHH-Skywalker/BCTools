import { defineStore } from "pinia"
import { ref, watch } from "vue"
import type { SongType } from "../api/types"
import { STORAGE_KEYS, readJSON, writeJSON } from "../utils/persist"

export type VacationBadge = "none" | "summer" | "winter"

interface AdvancedSettings {
  vacationBadge: VacationBadge
  posterBlurDorm: number
  posterBlurBroadcast: number
  posterQuoteDorm: string
  posterQuoteBroadcast: string
  simpleMode: boolean
  backgroundImageDorm: string | null
  backgroundImageBroadcast: string | null
}

const DEFAULT_BLUR = 12
const DEFAULT_QUOTE = "人生南北多歧路 君向潇湘我向秦"

function clampBlur(v: number): number {
  if (!Number.isFinite(v)) return DEFAULT_BLUR
  return Math.max(0, Math.min(50, Math.round(v)))
}

function loadAdvancedSettings(): AdvancedSettings {
  const parsed = readJSON<any>(STORAGE_KEYS.exportAdvanced)
  if (parsed) {
      // 兼容旧版本：旧的 backgroundImage 迁移到宿舍背景
      const legacyBg = typeof parsed.backgroundImage === "string" ? parsed.backgroundImage : null
      // 兼容旧版本：旧的 posterQuote 迁移到宿舍文案，旧的 posterBlur 迁移到宿舍模糊
      const legacyQuote = typeof parsed.posterQuote === "string" && parsed.posterQuote.trim() ? parsed.posterQuote : DEFAULT_QUOTE
      const legacyBlur = clampBlur(parsed.posterBlur)
      return {
        vacationBadge: ["none", "summer", "winter"].includes(parsed.vacationBadge) ? parsed.vacationBadge : "none",
        posterBlurDorm: clampBlur(parsed.posterBlurDorm ?? legacyBlur),
        posterBlurBroadcast: clampBlur(parsed.posterBlurBroadcast ?? legacyBlur),
        posterQuoteDorm: typeof parsed.posterQuoteDorm === "string" && parsed.posterQuoteDorm.trim() ? parsed.posterQuoteDorm : legacyQuote,
        posterQuoteBroadcast: typeof parsed.posterQuoteBroadcast === "string" && parsed.posterQuoteBroadcast.trim() ? parsed.posterQuoteBroadcast : legacyQuote,
        simpleMode: typeof parsed.simpleMode === "boolean" ? parsed.simpleMode : false,
        backgroundImageDorm: typeof parsed.backgroundImageDorm === "string" ? parsed.backgroundImageDorm : legacyBg,
        backgroundImageBroadcast: typeof parsed.backgroundImageBroadcast === "string" ? parsed.backgroundImageBroadcast : null,
      }
  }
  return {
    vacationBadge: "none",
    posterBlurDorm: DEFAULT_BLUR,
    posterBlurBroadcast: DEFAULT_BLUR,
    posterQuoteDorm: DEFAULT_QUOTE,
    posterQuoteBroadcast: DEFAULT_QUOTE,
    simpleMode: false,
    backgroundImageDorm: null,
    backgroundImageBroadcast: null,
  }
}

function saveAdvancedSettings(settings: AdvancedSettings) {
  writeJSON(STORAGE_KEYS.exportAdvanced, settings)
}

export const useExportStore = defineStore("export", () => {
  const selectedDates = ref<string[]>([])
  const customTemplateHTML = ref("")

  const advanced = ref<AdvancedSettings>(loadAdvancedSettings())
  const vacationBadge = ref<VacationBadge>(advanced.value.vacationBadge)
  const posterBlurDorm = ref<number>(advanced.value.posterBlurDorm)
  const posterBlurBroadcast = ref<number>(advanced.value.posterBlurBroadcast)
  const posterQuoteDorm = ref<string>(advanced.value.posterQuoteDorm)
  const posterQuoteBroadcast = ref<string>(advanced.value.posterQuoteBroadcast)
  const simpleMode = ref<boolean>(advanced.value.simpleMode)
  const backgroundImageDorm = ref<string | null>(advanced.value.backgroundImageDorm)
  const backgroundImageBroadcast = ref<string | null>(advanced.value.backgroundImageBroadcast)

  watch(vacationBadge, (val) => {
    advanced.value.vacationBadge = val
    saveAdvancedSettings(advanced.value)
  })

  watch(posterBlurDorm, (val) => {
    advanced.value.posterBlurDorm = clampBlur(val)
    saveAdvancedSettings(advanced.value)
  })

  watch(posterBlurBroadcast, (val) => {
    advanced.value.posterBlurBroadcast = clampBlur(val)
    saveAdvancedSettings(advanced.value)
  })

  watch(posterQuoteDorm, (val) => {
    advanced.value.posterQuoteDorm = val.trim() || DEFAULT_QUOTE
    saveAdvancedSettings(advanced.value)
  })

  watch(posterQuoteBroadcast, (val) => {
    advanced.value.posterQuoteBroadcast = val.trim() || DEFAULT_QUOTE
    saveAdvancedSettings(advanced.value)
  })

  watch(simpleMode, (val) => {
    advanced.value.simpleMode = val
    saveAdvancedSettings(advanced.value)
  })

  watch(backgroundImageDorm, (val) => {
    advanced.value.backgroundImageDorm = val
    saveAdvancedSettings(advanced.value)
  })

  watch(backgroundImageBroadcast, (val) => {
    advanced.value.backgroundImageBroadcast = val
    saveAdvancedSettings(advanced.value)
  })

  function setDates(dates: string[]) {
    selectedDates.value = [...new Set(dates)].sort()
  }

  function addDate(date: string) {
    if (!selectedDates.value.includes(date)) {
      selectedDates.value.push(date)
      selectedDates.value.sort()
    }
  }

  function removeDate(date: string) {
    selectedDates.value = selectedDates.value.filter((d) => d !== date)
  }

  function backgroundImageFor(type: SongType): string | null {
    return type === "dorm" ? backgroundImageDorm.value : backgroundImageBroadcast.value
  }

  function setBackgroundImage(type: SongType, value: string | null) {
    if (type === "dorm") backgroundImageDorm.value = value
    else backgroundImageBroadcast.value = value
  }

  return {
    selectedDates,
    customTemplateHTML,
    vacationBadge,
    posterBlurDorm,
    posterBlurBroadcast,
    posterQuoteDorm,
    posterQuoteBroadcast,
    simpleMode,
    backgroundImageDorm,
    backgroundImageBroadcast,
    advanced,
    setDates,
    addDate,
    removeDate,
    backgroundImageFor,
    setBackgroundImage,
  }
})
