import { beforeEach, describe, expect, it } from "vitest"
import { claimUpdateLog } from "./updateLog"
import type { UpdateLog } from "../api/types"

function update(overrides: Partial<UpdateLog> = {}): UpdateLog {
  return {
    schemaVersion: 1,
    id: "5.8.0.0-2026-09-15",
    version: "5.8.0.0",
    publishedAt: "2026-09-15T00:00:00Z",
    expiresAt: "2026-10-15T00:00:00Z",
    title: "Update",
    content: ["Item"],
    recent: true,
    startupId: "startup-1",
    ...overrides,
  }
}

describe("trusted update-log display", () => {
  beforeEach(() => localStorage.clear())

  it("claims a recent update once per installation storage", () => {
    expect(claimUpdateLog(update())).toBe(true)
    expect(claimUpdateLog(update())).toBe(false)
  })

  it("shows the same recent update again after a backend restart", () => {
    expect(claimUpdateLog(update())).toBe(true)
    expect(claimUpdateLog(update({ startupId: "startup-2" }))).toBe(true)
  })

  it("does not show an update the backend marked as older than 14 days", () => {
    expect(claimUpdateLog(update({ recent: false }))).toBe(false)
  })
})
