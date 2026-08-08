import apiClient from "./client"

export interface SystemStatus {
  version: string
  ffmpegAvailable: boolean
  ffprobeAvailable: boolean
}

export async function getSystemStatus(): Promise<SystemStatus> {
  const { data } = await apiClient.get<SystemStatus>("/system/status")
  return data
}
