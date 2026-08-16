import { describe, it, expect, vi, beforeEach } from "vitest"
import { ref } from "vue"
import { createPinia, setActivePinia } from "pinia"

// 记录每个文件进入后端处理的时刻与并发峰值。
const state = vi.hoisted(() => ({
  inFlight: 0,
  peak: 0,
  order: [] as string[],
  release: [] as Array<() => void>,
}))

vi.mock("@/api/files", () => ({
  processFile: vi.fn(async (f: File) => {
    state.inFlight++
    state.peak = Math.max(state.peak, state.inFlight)
    state.order.push(f.name)
    // 挂住，直到测试显式放行，这样才能观察到真实并发数
    await new Promise<void>((r) => state.release.push(r))
    state.inFlight--
    return { tempFileName: `${f.name}.mp3`, title: f.name, artist: "" }
  }),
  stashFile: vi.fn(async (f: File) => {
    state.inFlight++
    state.peak = Math.max(state.peak, state.inFlight)
    state.order.push(f.name)
    await new Promise<void>((r) => state.release.push(r))
    state.inFlight--
    return { tempFileName: `${f.name}.mp3`, title: f.name, artist: "" }
  }),
}))

vi.mock("@/api/songs", () => ({
  createSong: vi.fn(async (p: any) => ({ id: Math.floor(Math.random() * 1e6), ...p })),
  updateSong: vi.fn(async (_id: number, d: any) => d),
}))

import { useFileProcessor } from "@/composables/useFileProcessor"

const flush = () => new Promise((r) => setTimeout(r, 0))

describe("useFileProcessor concurrency", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    state.inFlight = 0
    state.peak = 0
    state.order = []
    state.release = []
  })

  it("processes up to `concurrency` files at once instead of serially", async () => {
    const date = ref("2026-08-20")
    const slot = ref<string | null>(null)
    const { handleFiles } = useFileProcessor(date, slot)

    const files = ["a.flac", "b.flac", "c.flac", "d.flac", "e.flac"].map(
      (n) => new File([new Uint8Array([1, 2, 3])], n),
    )
    handleFiles({ fileList: files })

    // 让已启动的链都推进到第一个 await
    await flush()
    await flush()

    // 回归点：此前 runQueue 只被调用一次且内部 await 后才递归，
    // 峰值恒为 1（完全串行），concurrency=3 形同废设。
    expect(state.peak).toBeGreaterThan(1)
    expect(state.peak).toBeLessThanOrEqual(3)

    // 放行全部，避免悬挂的 promise 泄漏到别的用例
    while (state.release.length) state.release.shift()!()
    await flush()
  })

  it("eventually processes every queued file", async () => {
    const date = ref("2026-08-20")
    const slot = ref<string | null>(null)
    const { handleFiles, processingFiles } = useFileProcessor(date, slot)

    const names = ["a.flac", "b.flac", "c.flac", "d.flac", "e.flac"]
    handleFiles({ fileList: names.map((n) => new File([new Uint8Array([1])], n)) })

    // 反复放行，直到所有文件都到达终态
    for (let i = 0; i < 60 && processingFiles.some((f) => f.status !== "done" && f.status !== "error"); i++) {
      while (state.release.length) state.release.shift()!()
      await flush()
    }

    expect(processingFiles).toHaveLength(5)
    expect(processingFiles.every((f) => f.status === "done")).toBe(true)
    // 每个文件都必须被真正处理过一次，不能有谁被并发改动漏掉
    expect(new Set(state.order).size).toBe(5)
  })
})
