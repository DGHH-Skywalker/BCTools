import { ref } from "vue"
import zhCN from "../locales/zh-CN.json"
import en from "../locales/en.json"

const messages: Record<string, any> = { "zh-CN": zhCN, en }
// 当前版本仅支持中文，保留 i18n 结构以便后续扩展
const currentLocale = ref<"zh-CN" | "en">("zh-CN")

export function useI18n() {
  const t = (key: string, params?: Record<string, any>): string => {
    const keys = key.split(".")
    let val: any = messages[currentLocale.value]
    for (const k of keys) {
      if (val == null) return key
      val = val[k]
    }
    if (typeof val !== "string") return key
    if (params) {
      return val.replace(/\{(\w+)\}/g, (_, k) => {
        return params[k] !== undefined ? String(params[k]) : `{${k}}`
      })
    }
    return val
  }

  const setLocale = (locale: "zh-CN" | "en") => {
    currentLocale.value = locale
    localStorage.setItem("locale", locale)
  }

  const weekdayName = (dayIndex: number): string => {
    return messages[currentLocale.value]?.weekdays?.[dayIndex - 1] ?? `day${dayIndex}`
  }

  const weekdayShortName = (dayIndex: number): string => {
    return messages[currentLocale.value]?.weekdaysShort?.[dayIndex - 1] ?? `${dayIndex}`
  }

  const weekdayLabel = (dayIndex: number): string => {
    const short = weekdayShortName(dayIndex)
    if (currentLocale.value === "zh-CN") {
      return `周${short}`
    }
    return short
  }

  const dayIndexFromDate = (dateStr: string): number => {
    const d = new Date(dateStr + "T00:00:00")
    const jsDay = d.getDay()
    return jsDay === 0 ? 7 : jsDay
  }

  return { t, setLocale, currentLocale, weekdayName, weekdayShortName, weekdayLabel, dayIndexFromDate }
}
