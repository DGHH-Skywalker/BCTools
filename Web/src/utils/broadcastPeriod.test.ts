import { describe, expect, it } from "vitest"
import { broadcastPeriod } from "./broadcastPeriod"

describe('broadcast period in China Standard Time', () => {
  it('preserves explicit periods, including ones inconsistent with creation time', () => {
    expect(broadcastPeriod({ period: 'noon', createdAt: '2026-09-18T07:00:00Z' })).toBe('noon')
  })
  it('does not classify 15:00 Beijing as morning just because UTC is 07:00', () => {
    expect(broadcastPeriod({ createdAt: '2026-09-18T07:00:00Z' })).toBe('afternoon')
    expect(broadcastPeriod({ createdAt: '2026-09-18T03:59:59Z' })).toBe('noon')
    expect(broadcastPeriod({ createdAt: '2026-09-18T04:00:00Z' })).toBe('afternoon')
    expect(broadcastPeriod({ createdAt: '' })).toBe('afternoon')
  })
})
