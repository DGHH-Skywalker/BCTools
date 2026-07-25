import { defineStore } from "pinia"
import { reactive } from "vue"
import type { ConversionStatus, ConversionFileType } from "../api/types"
import * as filesApi from "../api/files"

export interface ConversionState {
  id: string
  fileName: string
  fileType: ConversionFileType
  status: ConversionStatus
  progress: number
  songTitle: string
  songArtist: string
  tempFileName: string | null
  songId: number | null
  error: string | null
}

let counter = 0
function genId() { return `conv-${++counter}` }

export const useConversionStore = defineStore("conversion", () => {
  const files = reactive<Map<string, ConversionState>>(new Map())
  // Store actual File objects separately (Map is not ideal for reactive, but works with mutation)
  const fileMap = new Map<string, File>()

  function addFiles(fileList: File[]) {
    for (const file of fileList) {
      const ext = file.name.split(".").pop()?.toLowerCase() || ""
      const fileType: ConversionFileType = ["ncm", "mp3", "flac", "wav", "mp4"].includes(ext)
        ? (ext as ConversionFileType) : "mp3"
      const id = genId()
      fileMap.set(id, file)
      files.set(id, {
        id, fileName: file.name, fileType, status: "pending",
        progress: 0, songTitle: "", songArtist: "",
        tempFileName: null, songId: null, error: null,
      })
    }
  }

  async function processFile(id: string) {
    const state = files.get(id)
    if (!state) return
    state.status = "converting"
    try {
      const file = fileMap.get(id)
      if (!file) throw new Error("File not found")
      let result
      if (state.fileType === "ncm") {
        // Backend handles NCM decryption via unlock-music.cli
        result = await filesApi.processFile(file)
      } else if (state.fileType === "mp3") {
        result = await filesApi.stashFile(file)
      } else {
        // flac, wav, mp4 -> convert
        result = await filesApi.processFile(file)
      }
      state.status = "done"
      state.progress = 100
      state.songTitle = result.title || state.fileName.replace(/\.[^.]+$/, "")
      state.songArtist = result.artist || ""
      state.tempFileName = result.tempFileName
    } catch (err: any) {
      state.status = "error"
      state.error = err?.message || "处理失败"
    }
  }

  function updateProgress(id: string, progress: number) {
    const state = files.get(id)
    if (state) state.progress = progress
  }

  async function retryFile(id: string) {
    const state = files.get(id)
    if (!state) return
    state.status = "pending"
    state.error = null
    await processFile(id)
  }

  function removeFile(id: string) {
    files.delete(id)
    fileMap.delete(id)
  }

  function clearCompleted() {
    for (const [id, state] of files) {
      if (state.status === "done" || state.status === "error") {
        files.delete(id)
        fileMap.delete(id)
      }
    }
  }

  function getFile(id: string): File | undefined {
    return fileMap.get(id)
  }

  return { files, addFiles, processFile, updateProgress, retryFile, removeFile, clearCompleted, getFile }
})
