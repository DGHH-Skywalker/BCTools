import apiClient from "./client"

export async function backup(): Promise<void> {
  await apiClient.post("/sync/backup")
}
