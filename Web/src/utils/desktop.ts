export function runsInDesktop() {
  return location.hostname === "wails.localhost" || location.hostname.startsWith("wails.localhost.")
}

export function openDesktopWindow(view: string, args: Record<string, string> = {}) {
  const params = new URLSearchParams({ view, ...args })
  return fetch("/api/desktop/window?" + params, {
    method: "POST", headers: { "X-BCTools-Local": "1" },
  }).then(response => {
    if (!response.ok) throw new Error("无法打开桌面窗口")
  })
}

export function openUnlockWindow() {
  if (runsInDesktop()) return void openDesktopWindow("unlock").catch(console.error)
  window.open("/um-react/", "_blank")
}

export async function saveDownload(blob: Blob, name: string) {
  if (runsInDesktop()) {
    const response = await fetch("/api/desktop/save?name=" + encodeURIComponent(name), {
      method: "POST", headers: { "X-BCTools-Local": "1" }, body: blob,
    })
    if (!response.ok) throw new Error("无法保存文件")
    return Boolean((await response.json()).saved)
  }
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement("a")
  anchor.href = url
  anchor.download = name
  anchor.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
  return true
}
