import apiClient from "./client"
import type { Song, NewSong, SongType, ImportResult } from "./types"

export async function fetchSongs(type: SongType, dates?: string[]): Promise<Song[]> {
  const params: any = { type }
  if (dates && dates.length > 0) params.dates = dates.join(",")
  const res = await apiClient.get("/songs", { params })
  return res.data
}

export async function getSong(id: number): Promise<Song> {
  const res = await apiClient.get(`/songs/${id}`)
  return res.data
}

export async function createSong(data: NewSong): Promise<Song> {
  const res = await apiClient.post("/songs", data)
  return res.data
}

export async function updateSong(id: number, data: Partial<Song>): Promise<Song> {
  const res = await apiClient.put(`/songs/${id}`, data)
  return res.data
}

export async function deleteSong(id: number): Promise<void> {
  await apiClient.delete(`/songs/${id}`)
}

export async function sortSongs(type: SongType): Promise<Song[]> {
  const res = await apiClient.post(`/songs/sort?type=${type}`)
  return res.data
}

export async function reorderSongs(songIds: number[]): Promise<void> {
  await apiClient.post("/songs/reorder", { songIds })
}

export async function importXlsx(type: SongType, file: File): Promise<ImportResult> {
  const form = new FormData()
  form.append("file", file)
  const res = await apiClient.post(`/songs/import?type=${type}`, form, {
    headers: { "Content-Type": "multipart/form-data" },
  })
  return res.data
}

export async function exportXlsx(type: SongType, dates?: string[]): Promise<Blob> {
  const params: any = { type }
  if (dates && dates.length > 0) params.dates = dates.join(",")
  const res = await apiClient.get("/songs/export", { params, responseType: "blob" })
  return res.data
}
