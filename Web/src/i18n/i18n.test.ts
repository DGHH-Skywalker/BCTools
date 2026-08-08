import { describe, it, expect } from "vitest"
import { useI18n } from "./index"

describe("useI18n", () => {
  it("calculates dayIndex with Sunday as 7", () => {
    const { dayIndexFromDate } = useI18n()
    expect(dayIndexFromDate("2026-07-20")).toBe(1) // Monday
    expect(dayIndexFromDate("2026-07-25")).toBe(6) // Saturday
    expect(dayIndexFromDate("2026-07-26")).toBe(7) // Sunday
  })

  it("returns weekday names", () => {
    const { weekdayName, weekdayShortName } = useI18n()
    expect(weekdayName(1)).toBe("星期一")
    expect(weekdayShortName(7)).toBe("日")
  })
})
