import type { HotspotStatus, NetworkInfo } from "./types"

export async function getNetworkInfo(): Promise<NetworkInfo> {
  const res = await fetch("/api/network/info")
  if (!res.ok) throw new Error("获取网络信息失败")
  return res.json()
}

export async function startHotspot(): Promise<void> {
  const res = await fetch("/api/hotspot/start", { method: "POST" })
  if (!res.ok) throw new Error("启动热点失败")
}

export async function getHotspotStatus(): Promise<HotspotStatus> {
  const res = await fetch("/api/hotspot/status")
  if (!res.ok) throw new Error("获取热点状态失败")
  return res.json()
}
