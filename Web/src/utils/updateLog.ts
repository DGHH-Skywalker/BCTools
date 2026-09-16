import type { UpdateLog } from "../api/types"

const LAST_SHOWN_KEY = "bctools.update-log.last-shown"

export function claimUpdateLog(log: UpdateLog, storage: Storage = localStorage): boolean {
  if (!log.recent || !log.id) return false
  const value = `${log.startupId}:${log.id}`
  if (storage.getItem(LAST_SHOWN_KEY) === value) return false
  storage.setItem(LAST_SHOWN_KEY, value)
  return true
}
