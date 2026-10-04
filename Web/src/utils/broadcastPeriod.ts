import type { Song } from "../api/types"

export function broadcastPeriod(song: Pick<Song, "period" | "createdAt">): "noon" | "afternoon" {
  if (song.period === "noon" || song.period === "afternoon") return song.period
  const timestamp = Date.parse(song.createdAt)
  if (!Number.isFinite(timestamp) || timestamp <= 0) return "afternoon"
  return new Date(timestamp + 8 * 3600000).getUTCHours() < 12 ? "noon" : "afternoon"
}
