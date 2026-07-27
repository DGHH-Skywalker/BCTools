import { defineStore } from "pinia"
import { ref } from "vue"
import type { Song, NewSong, SongType } from "../api/types"
import * as songsApi from "../api/songs"

export const useSongsStore = defineStore("songs", () => {
  const dormSongs = ref<Song[]>([])
  const broadcastSongs = ref<Song[]>([])
  const loading = ref(false)

  async function fetchSongs(type: SongType, dates?: string[]) {
    loading.value = true
    try {
      const data = await songsApi.fetchSongs(type, dates)
      if (type === "dorm") dormSongs.value = data
      else broadcastSongs.value = data
    } finally {
      loading.value = false
    }
  }

  async function addSong(data: NewSong): Promise<Song> {
    const song = await songsApi.createSong(data)
    if (data.type === "dorm") dormSongs.value.push(song)
    else broadcastSongs.value.push(song)
    return song
  }

  async function updateSong(id: number, data: Partial<Song>): Promise<Song> {
    const song = await songsApi.updateSong(id, data)
    const list = dormSongs.value.find((s) => s.id === id) ? dormSongs : broadcastSongs
    const idx = list.value.findIndex((s) => s.id === id)
    if (idx >= 0) list.value[idx] = song
    return song
  }

  async function deleteSong(id: number): Promise<void> {
    await songsApi.deleteSong(id)
    dormSongs.value = dormSongs.value.filter((s) => s.id !== id)
    broadcastSongs.value = broadcastSongs.value.filter((s) => s.id !== id)
  }

  async function assignSong(songId: number, timeSlotId: string) {
    await updateSong(songId, { timeSlotId })
  }

  async function unassignSong(songId: number) {
    await updateSong(songId, { timeSlotId: null } as any)
  }

  return { dormSongs, broadcastSongs, loading, fetchSongs, addSong, updateSong, deleteSong, assignSong, unassignSong }
})
