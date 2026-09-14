import type { Song } from "../api/types"

const noiseWords = [
  "live", "cover", "伴奏", "原唱", "official", "mv", "ver", "version",
  "inst", "instrumental", "acoustic", "现场", "翻唱", "混音", "remix", "edition",
  "extended", "radio", "edit", "feat", "featuring", "ft", "ost",
]

function fullWidthToHalf(str: string): string {
  return str
    .replace(/[０-９]/g, (ch) => String.fromCharCode(ch.charCodeAt(0) - 0xFEE0))
    .replace(/[Ａ-Ｚ]/g, (ch) => String.fromCharCode(ch.charCodeAt(0) - 0xFEE0))
    .replace(/[ａ-ｚ]/g, (ch) => String.fromCharCode(ch.charCodeAt(0) - 0xFEE0))
}

export function normalizeTitle(title: string): string {
  let s = title.toLowerCase()
  s = fullWidthToHalf(s)
  // Remove bracketed content
  s = s.replace(/[\(\[\《\<｢〔].*?[\)\]\》\>｣〕]/g, " ")
  // Remove common noise words as whole words
  for (const w of noiseWords) {
    const pattern = new RegExp(`\\b${w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`, "g")
    s = s.replace(pattern, " ")
  }
  // Remove all non-word / non-CJK characters
  s = s.replace(/[^一-龥a-z0-9]/g, " ")
  // Collapse spaces
  s = s.replace(/\s+/g, " ").trim()
  return s
}

export function levenshtein(a: string, b: string): number {
  const m = a.length
  const n = b.length
  if (m === 0) return n
  if (n === 0) return m
  const prev = new Array(n + 1)
  const curr = new Array(n + 1)
  for (let j = 0; j <= n; j++) prev[j] = j
  for (let i = 1; i <= m; i++) {
    curr[0] = i
    for (let j = 1; j <= n; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1
      curr[j] = Math.min(curr[j - 1] + 1, prev[j] + 1, prev[j - 1] + cost)
    }
    for (let j = 0; j <= n; j++) prev[j] = curr[j]
  }
  return prev[n]
}

export function isSimilar(a: string, b: string): boolean {
  const na = normalizeTitle(a)
  const nb = normalizeTitle(b)
  if (!na || !nb) return false
  if (na === nb) return true
  if (na.length >= 2 && nb.includes(na)) return true
  if (nb.length >= 2 && na.includes(nb)) return true
  const maxLen = Math.max(na.length, nb.length)
  const threshold = Math.min(3, Math.floor(maxLen * 0.2))
  return levenshtein(na, nb) <= threshold
}

export function findSimilarSongs(
  title: string,
  date: string,
  allSongs: Song[],
  days: number,
  options: { includeSameDate?: boolean; excludeSongId?: number } = {},
): Song[] {
  if (!title || !date || days <= 0) return []
  const end = new Date(date)
  const start = new Date(end)
  start.setDate(start.getDate() - days)
  const matches: Song[] = []
  for (const song of allSongs) {
    if (!song.title) continue
    if (options.excludeSongId != null && song.id === options.excludeSongId) continue
    if (!options.includeSameDate && song.date === date) continue
    const d = new Date(song.date)
    const withinWindow = options.includeSameDate ? d >= start && d <= end : d >= start && d < end
    if (withinWindow && isSimilar(title, song.title)) {
      matches.push(song)
    }
  }
  return matches
}
