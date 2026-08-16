import { describe, it, expect } from "vitest"
import { ensureSegmenter, isSegmenterReady, segmentChineseTitle, formatExportTitle } from "@/utils/playlistTable"

describe("lazy Chinese segmenter", () => {
  it("degrades to the raw title before the dictionary is loaded", () => {
    expect(isSegmenterReady()).toBe(false)
    // 未加载时必须原样返回，不能抛错、不能返回空
    expect(segmentChineseTitle("稻香")).toBe("稻香")
  })

  it("actually segments once the dictionary is loaded", async () => {
    await ensureSegmenter()
    expect(isSegmenterReady()).toBe(true)
    const out = segmentChineseTitle("我们的歌")
    // 分词后会在词组间插入零宽空格 U+200B
    expect(out).toContain("\u200b")
    expect(out.replace(/\u200b/g, "")).toBe("我们的歌")
  })

  it("formatExportTitle escapes HTML and keeps segmentation", async () => {
    await ensureSegmenter()
    const out = formatExportTitle("<b>稻香</b>")
    expect(out).not.toContain("<b>")
    expect(out).toContain("&lt;")
  })
})
