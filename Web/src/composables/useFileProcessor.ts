import { reactive, computed } from "vue"
import { useSongsStore } from "../stores/songs"
import { useSettingsStore } from "../stores/settings"
import { processFile, stashFile } from "../api/files"
import { createSong, updateSong } from "../api/songs"
import { findSimilarSongs } from "../utils/songSimilarity"
import type { FileProcessResult, Song } from "../api/types"

export interface ProcessingFile {
  id: string
  file: File
  fileName: string
  fileType: string
  status: "pending" | "converting" | "done" | "error"
  songTitle: string
  tempFileName: string | null
  songId: number | null
  timeSlotId: string | null
  error: string
  duplicateWarnings?: Song[]
}

export function useFileProcessor(selectedDateRef: { value: string }, defaultTimeSlotIdRef?: { value: string | null | undefined }) {
  const songsStore = useSongsStore()
  const settingsStore = useSettingsStore()
  const processingFiles = reactive<ProcessingFile[]>([])
  const concurrency = 3
  let running = 0

  function getFileId(file: File): string {
    return `${file.name}|${file.size}|${file.lastModified}`
  }

  function isAlreadyProcessing(file: File): boolean {
    const id = getFileId(file)
    return processingFiles.some(
      (item) => item.status !== "error" && getFileId(item.file) === id
    )
  }

  function computeDuplicateWarnings(item: ProcessingFile) {
    if (!selectedDateRef.value || !item.songTitle) {
      item.duplicateWarnings = []
      return
    }
    item.duplicateWarnings = findSimilarSongs(
      item.songTitle,
      selectedDateRef.value,
      songsStore.dormSongs,
      settingsStore.duplicateCheckDays || 30
    )
  }

  const doneCount = computed(() => processingFiles.filter((f) => f.status === "done" || f.status === "error").length)

  function extractFiles(fileList: any[]): File[] {
    const files: File[] = []
    for (const f of fileList) {
      if (f instanceof File) {
        files.push(f)
      } else if (f && f.file instanceof File) {
        files.push(f.file)
      } else if (f && f.raw instanceof File) {
        files.push(f.raw)
      }
    }
    return files
  }

  function statusTagType(status: ProcessingFile["status"]): "success" | "error" | "warning" {
    if (status === "done") return "success"
    if (status === "error") return "error"
    return "warning"
  }

  function handleFiles({ fileList }: any): Promise<ProcessingFile[]> {
    const files = extractFiles(fileList || [])
    const added: ProcessingFile[] = []
    for (const file of files) {
      if (isAlreadyProcessing(file)) {
        // Skip files that are already pending/converting/done in this session.
        // This prevents the same file from being imported twice when the upload
        // component re-emits previously selected files.
        continue
      }
      const ext = file.name.split(".").pop()?.toLowerCase() || ""
      const item: ProcessingFile = {
        id: `f-${Date.now()}-${Math.random().toString(36).slice(2)}`,
        fileName: file.name,
        fileType: ext,
        file,
        status: "pending",
        songTitle: file.name.replace(/\.[^.]+$/, ""),
        tempFileName: null,
        songId: null,
        timeSlotId: null,
        error: "",
      }
      processingFiles.push(item)
      added.push(item)
    }
    runQueue()
    // Resolve once every file just added reaches a terminal status, so callers
    // (e.g. the um-react staged-import flow) can react to success/failure.
    return waitForSettled(added)
  }

  function waitForSettled(items: ProcessingFile[]): Promise<ProcessingFile[]> {
    return new Promise((resolve) => {
      const tick = () => {
        if (items.every((it) => it.status === "done" || it.status === "error")) {
          resolve(items)
        } else {
          setTimeout(tick, 80)
        }
      }
      tick()
    })
  }

  async function runQueue() {
    if (running >= concurrency) return
    const next = processingFiles.find((f) => f.status === "pending")
    if (!next) return
    running++
    await processItem(next)
    running--
    runQueue()
  }

  async function processItem(item: ProcessingFile) {
    item.status = "converting"
    item.error = ""
    const ext = item.fileType
    try {
      let result: FileProcessResult
      if (ext === "mp3") {
        result = await stashFile(item.file)
      } else {
        result = await processFile(item.file)
      }
      item.tempFileName = result.tempFileName
      item.songTitle = result.title || item.songTitle

      if (selectedDateRef.value && item.songTitle) {
        try {
          const payload: any = {
            type: "dorm",
            date: selectedDateRef.value,
            title: item.songTitle,
            filePath: result.tempFileName,
          }
          const defaultSlot = defaultTimeSlotIdRef?.value
          if (defaultSlot) {
            payload.timeSlotId = defaultSlot
          }
          const song = await createSong(payload)
          item.songId = song.id
          item.timeSlotId = defaultSlot || null
          songsStore.dormSongs.push(song)
        } catch (err: any) {
          item.status = "error"
          item.error = err?.message || "创建歌曲失败"
          return
        }
      }
      computeDuplicateWarnings(item)
      item.status = "done"
    } catch (err: any) {
      item.status = "error"
      item.error = err?.message || "处理失败"
      console.error("File processing error:", err)
    }
  }

  function updateTitle(item: ProcessingFile, v: string) {
    item.songTitle = v
    computeDuplicateWarnings(item)
    if (item.songId) {
      const storeSong = songsStore.dormSongs.find((s) => s.id === item.songId)
      if (storeSong) {
        storeSong.title = v
      }
    }
  }

  async function saveMetadata(item: ProcessingFile) {
    if (!item.songId) return
    try {
      await updateSong(item.songId, { title: item.songTitle })
      const storeSong = songsStore.dormSongs.find((s) => s.id === item.songId)
      if (storeSong) {
        storeSong.title = item.songTitle
      }
    } catch (err: any) {
      console.error("保存元数据失败", err)
    }
  }

  async function assignSlot(item: ProcessingFile, slotId: string): Promise<string | undefined> {
    if (!item.songId) return
    try {
      await songsStore.assignSong(item.songId, slotId)
      item.timeSlotId = slotId
    } catch (err: any) {
      return err?.message?.includes("已存在歌曲") ? "该时段已有歌曲，请更换时段或先删除原歌曲" : err?.message || "分配时段失败"
    }
  }

  function retryFile(item: ProcessingFile) {
    item.status = "pending"
    item.error = ""
    runQueue()
  }

  async function deleteFile(item: ProcessingFile): Promise<string | undefined> {
    try {
      if (item.songId) {
        await songsStore.deleteSong(item.songId)
      }
      const idx = processingFiles.indexOf(item)
      if (idx >= 0) processingFiles.splice(idx, 1)
    } catch (err: any) {
      return err?.message || "删除失败"
    }
  }

  function clearFiles() {
    processingFiles.length = 0
  }

  return {
    processingFiles,
    doneCount,
    statusTagType,
    handleFiles,
    retryFile,
    updateTitle,
    saveMetadata,
    assignSlot,
    deleteFile,
    clearFiles,
  }
}
