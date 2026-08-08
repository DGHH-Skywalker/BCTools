import apiClient from "./client"

export interface StageResult {
  stageId: string
  filename: string
  size: number
}

export interface StageMeta {
  filename: string
  size: number
  ext: string
  imported: boolean
}

// stageDecrypted uploads a decrypted audio blob to the backend so it can be
// imported by the main app from any device. Cross-page communication goes through
// the backend (not window.opener/postMessage) to support LAN/mobile access where
// the um-react tab and the main app may be on different devices.
export async function stageDecrypted(file: Blob, filename: string): Promise<StageResult> {
  const form = new FormData()
  form.append("file", file, filename)
  const res = await apiClient.post("/decrypt/stage", form, {
    headers: { "Content-Type": "multipart/form-data" },
    // Large audio files over a slow LAN may exceed the default 30s timeout.
    timeout: 0,
  })
  return res.data
}

export async function fetchStageMeta(stageId: string): Promise<StageMeta> {
  const res = await apiClient.get(`/decrypt/stage/${stageId}`)
  return res.data
}

export async function fetchStageFile(stageId: string): Promise<Blob> {
  const res = await apiClient.get(`/decrypt/stage/${stageId}/file`, { responseType: "blob" })
  return res.data
}

export async function deleteStage(stageId: string): Promise<void> {
  await apiClient.delete(`/decrypt/stage/${stageId}`)
}

// markStageImported tells the backend the staged file was successfully imported.
// The um-react tab polls the stage meta; seeing imported=true lets it remove the
// decrypted card via its own delete logic. The stage is left in place so um-react
// can read the flag, then um-react deletes it after removing the card.
export async function markStageImported(stageId: string): Promise<void> {
  await apiClient.post(`/decrypt/stage/${stageId}/imported`)
}
