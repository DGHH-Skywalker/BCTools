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
  entries: { source: string; sources?: string[]; targetName: string }[]
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

// fetchSilentMP3 让后端返回固定 17.43s 的静音 MP3。
//
// 重要：时长由后端 converter.DefaultSilentPlaceholder 写死——前端的
// File System Access API 那条导出路径在这里和后端内部 organizeService 走的是
// 同一个常量，SD 卡上同一序号文件的时长才会一致。**绝不要让前端传时长**，
// 否则会出现「整天空着时后端 internal 路径 17.43s、浏览器路径 30s」这种对不上
// 的诡异 bug。
export async function fetchSilentMP3(): Promise<Blob> {
  const res = await fetch("/api/files/silent")
  if (!res.ok) throw new Error("生成静音文件失败")
  return res.blob()
}

// fetchMergedMP3 让后端把同一时段的多首歌按顺序合并成一个 MP3 并回传。
// 用于「文件系统访问 API」那条导出路径——那条路径由浏览器自己写 SD 卡，
// 拿不到服务端内部的合并结果。
export async function fetchMergedMP3(sources: string[]): Promise<Blob> {
  const res = await fetch("/api/files/merge", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ sources }),
  })
  if (!res.ok) throw new Error("合并音频失败")
  return res.blob()
}

export async function deleteSourceFile(filename: string): Promise<void> {
  await apiClient.delete("/files/source", { params: { file: filename } })
}

export async function browseDir(dir: string): Promise<{ path: string; dirs: string[]; files: string[] }> {
  const res = await apiClient.get("/files/browse", { params: { dir } })
  return res.data
}
