import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

describe('title save queue', () => {
  beforeEach(() => { vi.resetModules(); vi.useFakeTimers(); localStorage.clear() })
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals() })
  it('saves without blur and clears only after the server acknowledges', async () => {
    const saves = await import('./pendingSaves')
    const save = vi.fn(async () => {})
    saves.queueTitle(1, '刚录入', save)
    expect(saves.unsavedCount.value).toBe(1)
    expect(localStorage.length).toBe(1)
    await vi.advanceTimersByTimeAsync(400)
    expect(save).toHaveBeenCalledWith('刚录入')
    expect(saves.unsavedCount.value).toBe(0)
  })
  it('serializes edits made while an earlier request is pending', async () => {
    const saves = await import('./pendingSaves')
    let resolve!: () => void
    const save = vi.fn().mockImplementationOnce(() => new Promise<void>(r => { resolve = r })).mockResolvedValue(undefined)
    saves.queueTitle(2, '旧输入', save)
    const pending = saves.flushTitle(2)
    saves.queueTitle(2, '最新输入', save)
    resolve()
    await pending
    expect(save.mock.calls.map(args => args[0])).toEqual(['旧输入', '最新输入'])
    expect(saves.unsavedCount.value).toBe(0)
  })
  it('retains failed drafts and refuses a handoff until retry succeeds', async () => {
    const saves = await import('./pendingSaves')
    const save = vi.fn().mockRejectedValueOnce(new Error('disk failed')).mockResolvedValue(undefined)
    saves.queueTitle(3, '不能丢', save)
    await expect(saves.flushAllSaves()).rejects.toThrow()
    expect(saves.unsavedCount.value).toBe(1)
    expect(localStorage.length).toBe(1)
    expect(saves.saveError.value).not.toBe('')
    await saves.flushAllSaves()
    expect(saves.unsavedCount.value).toBe(0)
  })
  it('finds a previous page backup only for the same fixed database', async () => {
    const oldPage = await import('./pendingSaves')
    oldPage.databaseID.value = 'database-A'
    oldPage.queueTitle(9, '刷新不能丢', vi.fn(async () => {}), { type: 'broadcast', originalTitle: '原歌名', date: '2026-10-04', createdAt: '2026-10-04T00:00:00Z' })
    vi.resetModules()
    const newPage = await import('./pendingSaves')
    newPage.databaseID.value = 'database-B'
    expect(newPage.recoverableBackups()).toEqual([])
    newPage.databaseID.value = 'database-A'
    const backup = newPage.recoverableBackups()[0]
    expect(backup.drafts[0].title).toBe('刷新不能丢')
    expect(backup.drafts[0].context?.originalTitle).toBe('原歌名')
    newPage.discardBackup(backup.key)
    expect(newPage.recoverableBackups()).toEqual([])
  })
  it('opens on LAN HTTP even when randomUUID is unavailable', async () => {
    vi.stubGlobal('crypto', {})
    const saves = await import('./pendingSaves')
    expect(saves.browserClientID.length).toBeGreaterThan(8)
  })
})
