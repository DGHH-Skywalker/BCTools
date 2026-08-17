import apiClient from "./client"

export type MigrationScope = "json" | "files"

export interface MigrationPreview {
  backupDir: string
  exists: boolean
  existingFiles?: string[]
  willOverwrite: number
  collisionIndex: number
}

export interface MigrationExportResult {
  backupDir?: string
  writtenFiles?: string[]
  skippedFiles?: string[]
  songCount?: number
  // 后端在确认覆盖那一步用这个标记，让前端弹框
  confirmNeeded?: boolean
  existingFiles?: string[]
}

/**
 * POST /api/migration/preview — 给前端问「下一个候选子目录是否已存在」。
 */
export async function previewMigration(targetDir: string): Promise<MigrationPreview> {
  const res = await apiClient.post("/migration/preview", { targetDir })
  return res.data
}

/**
 * POST /api/migration/export — 真正落盘。
 *
 *   - scope="json"  走 Content-Disposition=attachment，浏览器自动下载单 JSON 文件。
 *   - scope="files" 走两阶段：先看响应 confirmNeeded；为 true 就弹框让用户选
 *     「换文件夹 / 覆盖」；选覆盖再带 confirm=true 重发。
 *
 * 注意：json 模式的响应不是 JSON 体，而是 application/json 文件流；这里在函数内
 * 直接做 blob 触发下载，调用方拿不到返回值。
 */
export async function exportMigrationJson(): Promise<void> {
  const res = await apiClient.post(
    "/migration/export",
    { scope: "json" },
    { responseType: "blob" },
  )
  const blob = res.data as Blob
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  a.download = "broadcast-tool-data.json"
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  // 留一拍再 revoke，让浏览器有足够时间开始下载
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export async function exportMigrationFiles(
  targetDir: string,
  confirm = false,
): Promise<MigrationExportResult> {
  const res = await apiClient.post("/migration/export", {
    targetDir,
    scope: "files",
    confirm,
  })
  return res.data
}

export interface MigrationImportResult {
  insertedDorm: number
  insertedBroadcast: number
  filesCopied: number
  filesMissing: string[]
  timeSlotsMerged: number
  skippedEmpty: number
}

/**
 * POST /api/migration/import — 从备份目录或 JSON 文件还原数据。
 *
 *   - backupDir 走"目录导入"：之前 export 出的 `xxxx年xx月xx日广播站点歌工具数据备份1/`
 *   - jsonData 走"文件导入"：浏览器里读好整文件传进来（不带音频）
 *   - mode: "merge"（默认，追加不覆盖）/ "replace"（先清空本机歌单与时段再导入）
 */
export async function importMigration(req: {
  backupDir?: string
  jsonData?: any
  mode?: "merge" | "replace"
}): Promise<MigrationImportResult> {
  const res = await apiClient.post("/migration/import", req)
  return res.data
}
