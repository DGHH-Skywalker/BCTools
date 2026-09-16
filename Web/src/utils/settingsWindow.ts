const SETTINGS_WINDOW_NAME = "bctools-settings"

export function openSettingsWindow(path = "/settings") {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`
  const url = `${window.location.origin}${window.location.pathname}#${normalizedPath}`
  const settingsWindow = window.open(url, SETTINGS_WINDOW_NAME)
  settingsWindow?.focus()
}
