import { ref } from "vue"
import type { SongType } from "../api/types"
import { readDraftBackups, writeDraftBackup } from "./persist"
import { version as frontendVersion } from "../../package.json"

export type DraftContext = { type: SongType; originalTitle: string; date: string; createdAt: string }
export type StoredDraft = { id: number; title: string; context?: DraftContext }
export type DraftBackup = { key: string; databaseId: string; clientId: string; savedAt: string; drafts: StoredDraft[] }
type Draft = StoredDraft & { save: (title: string) => Promise<unknown> }
const drafts = new Map<number, Draft>()
const running = new Map<number, Promise<void>>()
const timers = new Map<number, ReturnType<typeof setTimeout>>()
export const unsavedCount = ref(0)
export const saveError = ref("")
export const handoffFrozen = ref(false)
export const activeMutations = ref(0)
export const databaseID = ref("")
export const browserClientID = globalThis.crypto?.randomUUID?.() || `client-${Date.now()}-${Math.random().toString(36).slice(2)}`
const recoveryKey = `bctools.unsaved-titles.${browserClientID}`

function changed() {
  unsavedCount.value = drafts.size
  try {
    writeDraftBackup(recoveryKey, drafts.size ? {
      databaseId: databaseID.value, clientId: browserClientID, savedAt: new Date().toISOString(),
      drafts: [...drafts.values()].map(({ id, title, context }) => ({ id, title, context })),
    } : null)
  } catch { saveError.value = "本地草稿备份失败，请勿关闭页面" }
}

export function queueTitle(id: number, title: string, save: Draft["save"], context?: DraftContext) {
  if (handoffFrozen.value) return
  drafts.set(id, { id, title, save, context: drafts.get(id)?.context || context })
  changed()
  clearTimeout(timers.get(id))
  timers.set(id, setTimeout(() => { void flushTitle(id).catch(() => {}) }, 400))
}

export function flushTitle(id: number): Promise<void> {
  clearTimeout(timers.get(id)); timers.delete(id)
  const pending = running.get(id)
  if (pending) return pending
  const task = (async () => {
    while (drafts.has(id)) {
      const draft = drafts.get(id)!
      try { await draft.save(draft.title) }
      catch (e) { saveError.value = "歌名尚未保存，草稿已保留；请重试，暂勿退出"; throw e }
      // A slower response must not delete a newer edit.
      if (drafts.get(id) === draft) drafts.delete(id)
      changed()
    }
    if (!drafts.size) saveError.value = ""
  })().finally(() => running.delete(id))
  running.set(id, task)
  return task
}

export async function flushAllSaves() {
  const deadline = Date.now() + 45000
  do {
    await Promise.all([...drafts.keys()].map(flushTitle))
    if (!activeMutations.value && !drafts.size) return
    if (Date.now() > deadline) throw new Error("其他操作尚未保存")
    await new Promise(resolve => setTimeout(resolve, 50))
  } while (true)
}

export function recoverableBackups(): DraftBackup[] {
  try {
    return readDraftBackups().flatMap(({ key, value }) => {
      const backup = value as Partial<DraftBackup> | null
      if (!backup || key === recoveryKey || backup.databaseId !== databaseID.value || !Array.isArray(backup.drafts)) return []
      if (!backup.drafts.every(d => Number.isSafeInteger(d.id) && typeof d.title === "string")) return []
      return [{ ...backup, key } as DraftBackup]
    })
  } catch { saveError.value = "无法读取本地草稿，请勿清理浏览器数据"; return [] }
}
export function discardBackup(key: string) { writeDraftBackup(key, null) }

export async function reportSaveState(ack = "", leave = false) {
  const res = await fetch('/api/lifecycle/client', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ id: browserClientID, ack, leave, version: frontendVersion, dirty: !!(unsavedCount.value || activeMutations.value) }),
  })
  if (!res.ok) throw new Error('无法登记页面保存状态')
  const state = await res.json()
  if (state.databaseId) databaseID.value = state.databaseId
  return state as { prepare: string; frozen: boolean; databaseId: string; instanceId: string; clients: string[] }
}
