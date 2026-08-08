import apiClient from "./client"
import type { FileProcessResult, OrganizeResult } from "./types"

export async function processFile(file: File): Promise<FileProcessResult> {
  const form = new FormData()
  form.append("file", file)
  const res = await apiClient.post("/files/process", form, {
    headers: { "Content-Type": "multipart/form-data" },
    onUploadProgress: (e) => { if (e.total) console.log(Math.round((e.loaded / e.total) * 100)) },
  })
  return res.data
}

export async function stashFile(file: File): Promise<FileProcessResult> {
  const form = new FormData()
  form.append("file", file)
  const res = await apiClient.post("/files/stash", form, {
    headers: { "Content-Type": "multipart/form-data" },
  })
  return res.data
}

export async function organizeFiles(data: {
  entries: { source: string; targetName: string }[]
  targetDir: string
  mode: "copy" | "move"
  confirm?: boolean
}): Promise<OrganizeResult> {
  const res = await apiClient.post("/files/organize", data)
  return res.data
}

export async function selectDir(): Promise<string> {
  const res = await apiClient.post("/files/select-dir")
  return res.data.path
}

export async function fetchSilentMP3(duration: number): Promise<Blob> {
  const res = await fetch(`/api/files/silent?duration=${duration}`)
  if (!res.ok) throw new Error("生成静音文件失败")
  return res.blob()
}

export async function deleteSourceFile(filename: string): Promise<void> {
  await apiClient.delete("/files/source", { params: { file: filename } })
}

export async function browseDir(dir: string): Promise<{ path: string; dirs: string[]; files: string[] }> {
  const res = await apiClient.get("/files/browse", { params: { dir } })
  return res.data
}
