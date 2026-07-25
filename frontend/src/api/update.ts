import apiClient from "./client"
import type { UpdateCheckResult } from "./types"

export async function checkUpdate(): Promise<UpdateCheckResult> {
  const res = await apiClient.get("/check-update")
  return res.data
}
