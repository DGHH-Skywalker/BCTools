import { describe, expect, it } from "vitest"
import { compatibleAdvancedSettings, normalizeAdvancedSettings } from "./export"

describe("playlist background compatibility", () => {
  it("inherits single-background settings from older releases", () => {
    const settings = normalizeAdvancedSettings({
      backgroundImage: "data:image/png;base64,legacy",
      posterBlur: 8,
      posterQuote: "旧版文案",
    })

    expect(settings.backgroundImageDorm).toBe("data:image/png;base64,legacy")
    expect(settings.posterBlurDorm).toBe(8)
    expect(settings.posterQuoteDorm).toBe("旧版文案")
  })

  it("writes aliases that older releases can read", () => {
    const settings = normalizeAdvancedSettings({
      backgroundImageDorm: "data:image/png;base64,current",
      posterBlurDorm: 10,
      posterQuoteDorm: "新版文案",
    })
    const compatible = compatibleAdvancedSettings(settings)

    expect(compatible.backgroundImage).toBe(settings.backgroundImageDorm)
    expect(compatible.posterBlur).toBe(settings.posterBlurDorm)
    expect(compatible.posterQuote).toBe(settings.posterQuoteDorm)
  })
})
