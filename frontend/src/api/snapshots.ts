import apiClient from "./client"
import type { SnapshotInfo } from "./types"

export async function getSnapshots(): Promise<SnapshotInfo[]> {
  const res = await apiClient.get("/snapshots")
  return res.data
}

export async function restoreSnapshot(filename: string): Promise<void> {
  await apiClient.post(`/snapshots/restore?filename=${filename}`)
}
