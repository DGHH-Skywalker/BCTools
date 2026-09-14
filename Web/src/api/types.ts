export type SongType = "dorm" | "broadcast"
export type ConversionStatus = "pending" | "converting" | "done" | "error"
export type ConversionFileType = "ncm" | "mp3" | "flac" | "wav" | "mp4"

export interface Song {
  id: number
  date: string
  weekday: string
  title: string
  artist: string
  remark: string
  filePath: string
  timeSlotId: string | null
  createdAt: string
  period?: "noon" | "afternoon"
  order?: number
}

export interface NewSong {
  type: SongType
  date: string
  title: string
  artist?: string
  remark?: string
  filePath?: string
  timeSlotId?: string | null
  period?: "noon" | "afternoon"
}

export interface TimeSlot {
  id: string
  dayIndex: number
  order: number
  time: string
}

export interface Settings {
  timeSlots: TimeSlot[]
  allowTemplateJS: boolean
  version: string
  downloadUrl: string
  adminPasswordHint: string
  locale: string
  broadcastColumnMap: Record<string, string>
  duplicateCheckDays: number
}

export interface NetworkInfo {
  ip: string
  port: number
  url: string
  isWindows: boolean
  windowsVersion: string
  isLocalRequest: boolean
}

export interface ImportResult {
  inserted: number
  skipped: number
  errors: { row: number; message: string }[]
}

export interface FileProcessResult {
  tempFileName: string
  title: string
  artist: string
}

export interface OrganizeResult {
  successful: { source: string; target: string }[]
  failed: { source: string; reason: string }[]
  warning?: string
  confirmNeeded?: boolean
  existingFiles?: string[]
}

export interface ExportWeekResult {
  confirmNeeded?: boolean
  existingFiles?: string[]
  targetDir?: string
  fileCount?: number
}

export interface UpdateCheckResult {
  hasUpdate: boolean
  latestVersion?: string
  downloadUrl?: string
}

export interface SnapshotInfo {
  filename: string
  time: string
  size: number
}
