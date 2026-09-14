import { useSettingsStore } from "../stores/settings"
import { useSongsStore } from "../stores/songs"
import { findSimilarSongs } from "../utils/songSimilarity"
import type { Song, SongType } from "../api/types"

/** Provides the single cross-playlist duplicate-warning policy used by playlist editors. */
export function useSongDuplicateWarnings(_type: SongType) {
  const songsStore = useSongsStore()
  const settingsStore = useSettingsStore()

  function findWarnings(song: Pick<Song, "id" | "title" | "date">, source?: Song[]): Song[] {
    // 两份歌单共用同一个播出资源，重复提醒必须跨歌单生效。
    const songs = source || [...songsStore.dormSongs, ...songsStore.broadcastSongs]
    return findSimilarSongs(song.title, song.date, songs, settingsStore.duplicateCheckDays || 30, {
      includeSameDate: true,
      excludeSongId: song.id,
    })
  }

  return { findWarnings }
}
