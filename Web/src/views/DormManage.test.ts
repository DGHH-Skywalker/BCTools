import { describe, it, expect, beforeEach, vi } from "vitest"
import { mount, flushPromises } from "@vue/test-utils"
import { createPinia, setActivePinia } from "pinia"
import { defineComponent, h } from "vue"
import { createRouter, createMemoryHistory } from "vue-router"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"
import naive from "naive-ui"
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from "naive-ui"

// Mock songs API
vi.mock("@/api/songs", () => {
  const today = new Date()
  const nextWeek = new Date(today.getTime() + 7 * 24 * 60 * 60 * 1000)
  const y = nextWeek.getFullYear()
  const m = String(nextWeek.getMonth() + 1).padStart(2, "0")
  const d = String(nextWeek.getDate()).padStart(2, "0")
  const nextWeekStr = `${y}-${m}-${d}`
  const jsDay = nextWeek.getDay()
  const weekday = jsDay === 0 ? "周日" : ["周日","周一","周二","周三","周四","周五","周六"][jsDay]
  return {
    fetchSongs: vi.fn(async (type: string) => {
      if (type === "dorm") {
        return [
          { id: 1, date: nextWeekStr, weekday, title: "测试歌曲A", artist: "", remark: "", filePath: "/songs/a.mp3", timeSlotId: "slot-1", createdAt: "" },
          { id: 2, date: nextWeekStr, weekday, title: "测试歌曲B", artist: "", remark: "", filePath: "", timeSlotId: null, createdAt: "" },
        ]
      }
      return []
    }),
    createSong: vi.fn(async (d: any) => ({ id: Date.now(), ...d })),
    updateSong: vi.fn(async (id: number, d: any) => ({ id, ...d })),
    deleteSong: vi.fn(async () => {}),
    sortSongs: vi.fn(async () => []),
    getSong: vi.fn(async () => ({})),
    importXlsx: vi.fn(async () => ({})),
    exportXlsx: vi.fn(async () => new Blob()),
  }
})

// Mock settings API（设置相关端点已并入 api/misc）
vi.mock("@/api/misc", () => ({
  getSettings: vi.fn(async () => ({
    timeSlots: [{ id: "slot-1", dayIndex: 1, time: "12:00", order: 1 }],
    allowTemplateJS: false,
    version: "5.5.0.0",
    downloadUrl: "",
    adminPasswordHint: "",
    locale: "zh-CN",
    broadcastColumnMap: {},
    duplicateCheckDays: 30,
  })),
  updateSettings: vi.fn(async (d: any) => d),
}))

// Mock useAudioPlayer
vi.mock("@/composables/useAudioPlayer", () => ({
  useAudioPlayer: () => ({
    currentFile: { value: null },
    currentTitle: { value: "" },
    isPlaying: { value: false },
    isVisible: { value: false },
    currentTime: { value: 0 },
    duration: { value: 0 },
    error: { value: "" },
    play: vi.fn(),
    pause: vi.fn(),
    stop: vi.fn(),
    toggle: vi.fn(),
    show: vi.fn(),
    hide: vi.fn(),
  }),
  formatDuration: (s: number) => "00:00",
}))

// Mock vuedraggable: render the #item slot for each item, no SortableJS in jsdom.
vi.mock("vuedraggable", () => ({
  default: defineComponent({
    name: "Draggable",
    props: ["modelValue", "itemKey", "handle", "animation", "disabled"],
    emits: ["update:modelValue"],
    setup(props, { slots }) {
      return () => {
        const items = (props.modelValue as any[]) || []
        return h("div", items.map((item) => slots.item?.({ element: item })))
      }
    },
  }),
}))

import DormManage from "./DormManage.vue"

dayjs.extend(isoWeek)

describe("DormManage (integrated page)", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it("mounts and renders songs for the next week without throwing", async () => {
    const errors: string[] = []
    const origError = console.error
    console.error = (...args: any[]) => {
      errors.push(args.map(String).join(" "))
      // don't spam
    }

    const Wrapper = defineComponent({
      setup() {
        return () =>
          h(
            NConfigProvider,
            null,
            {
              default: () =>
                h(
                  NMessageProvider,
                  null,
                  {
                    default: () =>
                      h(
                        NDialogProvider,
                        null,
                        {
                          default: () =>
                            h(NNotificationProvider, null, { default: () => h(DormManage) }),
                        },
                      ),
                  },
                ),
            },
          )
      },
    })

    const wrapper = mount(Wrapper, {
      global: {
        plugins: [
          naive,
          createRouter({
            history: createMemoryHistory(),
            routes: [{ path: "/dorm/manage", component: DormManage }],
          }),
        ],
      },
    })

    await flushPromises()
    await flushPromises()

    console.error = origError

    const html = wrapper.html()
    // next week's songs should render
    expect(html).toContain("测试歌曲A")
    expect(html).toContain("测试歌曲B")
    // no Vue runtime errors
    const vueErrors = errors.filter((e) => e.includes("TypeError") || e.includes("is not") || e.includes("Failed to resolve"))
    expect(vueErrors.length).toBe(0)
  })
})
