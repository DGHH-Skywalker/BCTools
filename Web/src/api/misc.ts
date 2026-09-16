// 杂项 API：把只有一两个函数、彼此平行的小模块收在一处，
// 避免为每个端点单开一个文件。
//
// 领域较重的 API 仍各自独立成文件（songs / files / decrypt）。
import apiClient from "./client"
import type { NetworkInfo, Settings, UpdateLog } from "./types"

// ---------- 设置 ----------

export async function getSettings(): Promise<Settings> {
  const res = await apiClient.get("/settings")
  return res.data
}

export async function updateSettings(
  data: Partial<Settings> & { confirmed?: boolean },
): Promise<Settings> {
  const res = await apiClient.put("/settings", data)
  return res.data
}

// ---------- 网络 / 局域网访问 ----------

export async function getNetworkInfo(): Promise<NetworkInfo> {
  const res = await apiClient.get("/network/info")
  return res.data
}

// ---------- 系统状态 ----------

export interface SystemStatus {
  version: string
  ffmpegAvailable: boolean
  ffprobeAvailable: boolean
}

export async function getSystemStatus(): Promise<SystemStatus> {
  const { data } = await apiClient.get<SystemStatus>("/system/status")
  return data
}

// ---------- 可信更新日志 ----------

export async function getLatestUpdateLog(): Promise<UpdateLog> {
  const { data } = await apiClient.get<UpdateLog>("/update-log/latest")
  return data
}
