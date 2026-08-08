import type { NetworkInfo } from "./types"

export async function getNetworkInfo(): Promise<NetworkInfo> {
  const res = await fetch("/api/network/info")
  if (!res.ok) throw new Error("获取网络信息失败")
  return res.json()
}
