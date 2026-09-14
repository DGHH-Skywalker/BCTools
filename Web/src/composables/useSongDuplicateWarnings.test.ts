import { beforeEach, describe, expect, it } from "vitest"
import { createPinia, setActivePinia } from "pinia"
import { useSongDuplicateWarnings } from "./useSongDuplicateWarnings"
import { useSettingsStore } from "../stores/settings"
import { useSongsStore } from "../stores/songs"
import type { Song } from "../api/types"

function song(id: number, date: string, title: string): Song {
  return {
    id,
    date,
    weekday: "",
    title,
    artist: "",
    remark: "",
    filePath: "",
    timeSlotId: null,
    createdAt: "",
  }
}

describe("useSongDuplicateWarnings", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    useSettingsStore().duplicateCheckDays = 30
  })

  it("warns from both playlists when editing a dorm song, including same-day broadcast songs", () => {
    const songsStore = useSongsStore()
    songsStore.dormSongs = [song(1, "2026-07-01", "晴天"), song(3, "2026-07-10", "晴天")]
    songsStore.broadcastSongs = [song(2, "2026-07-10", "晴天")]

    expect(useSongDuplicateWarnings("dorm").findWarnings(song(3, "2026-07-10", "晴天")))
      .toEqual([songsStore.dormSongs[0], songsStore.broadcastSongs[0]])
  })

  it("uses the same cross-playlist policy when editing a broadcast song", () => {
    const songsStore = useSongsStore()
    songsStore.dormSongs = [song(1, "2026-07-10", "晴天")]
    songsStore.broadcastSongs = [song(2, "2026-07-02", "晴天"), song(3, "2026-07-10", "晴天")]

    expect(useSongDuplicateWarnings("broadcast").findWarnings(song(3, "2026-07-10", "晴天")))
      .toEqual([songsStore.dormSongs[0], songsStore.broadcastSongs[0]])
  })
})
