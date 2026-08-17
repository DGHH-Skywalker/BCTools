import { ref, computed } from "vue"
import { darkTheme } from "naive-ui"
import { COLORS } from "../constants/colors"

export interface ThemeOverrides {
  common: {
    primaryColor: string
    primaryColorHover: string
    primaryColorPressed: string
    primaryColorSuppl: string
  }
}

const themeOverrides = ref<ThemeOverrides>({
  common: {
    primaryColor: COLORS.primary,
    primaryColorHover: COLORS.primaryHover,
    primaryColorPressed: COLORS.primaryPressed,
    primaryColorSuppl: COLORS.primary,
  },
})

const isDark = ref<boolean>(
  typeof window !== "undefined" && window.matchMedia
    ? window.matchMedia("(prefers-color-scheme: dark)").matches
    : false,
)

const theme = computed(() => (isDark.value ? darkTheme : null))

if (typeof window !== "undefined" && window.matchMedia) {
  const mq = window.matchMedia("(prefers-color-scheme: dark)")
  const listener = (e: MediaQueryListEvent) => {
    isDark.value = e.matches
  }
  if (mq.addEventListener) {
    mq.addEventListener("change", listener)
  } else if ((mq as any).addListener) {
    ;(mq as any).addListener(listener)
  }
}

function hexToRgb(hex: string) {
  const v = hex.replace("#", "")
  const full = v.length === 3 ? v.split("").map((c) => c + c).join("") : v
  const num = parseInt(full, 16)
  return { r: (num >> 16) & 255, g: (num >> 8) & 255, b: num & 255 }
}

function rgbToHex(r: number, g: number, b: number) {
  return "#" + [r, g, b].map((x) => Math.max(0, Math.min(255, x)).toString(16).padStart(2, "0")).join("")
}

function adjustColor(hex: string, amount: number) {
  const { r, g, b } = hexToRgb(hex)
  return rgbToHex(r + amount, g + amount, b + amount)
}

export function applyTheme(color: string) {
  themeOverrides.value = {
    common: {
      primaryColor: color,
      primaryColorHover: adjustColor(color, 25),
      primaryColorPressed: adjustColor(color, -25),
      primaryColorSuppl: color,
    },
  }
  document.documentElement.style.setProperty("--theme-color", color)
  document.documentElement.style.setProperty("--theme-color-light", adjustColor(color, 60) + "40")
}

export function useTheme() {
  return { themeOverrides, theme, isDark, applyTheme, adjustColor }
}
