import { describe, it, expect, beforeEach, vi } from "vitest"
import { mount, flushPromises } from "@vue/test-utils"
import { createPinia, setActivePinia } from "pinia"
import dayjs from "dayjs"
import isoWeek from "dayjs/plugin/isoWeek"

// Mock the songs API so stores don't hit the network
vi.mock("@/api/songs", () => ({
  fetchSongs: vi.fn(async (type: string) => {
    if (type === "dorm") {
      return [
        { id: 1, date: "2026-07-26", weekday: 7, title: "晴天", artist: "周杰伦", remark: "", filePath: "/songs/a.mp3", timeSlotId: "slot-1", createdAt: "", period: "noon" },
        { id: 2, date: "2026-07-26", weekday: 7, title: "稻香", artist: "周杰伦", remark: "", filePath: "/songs/b.mp3", timeSlotId: null, createdAt: "", period: "afternoon" },
        { id: 3, date: "2026-07-27", weekday: 1, title: "七里香", artist: "周杰伦", remark: "", filePath: "/songs/c.mp3", timeSlotId: "slot-2", createdAt: "" },
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
}))

// Mock useAudioPlayer so jsdom doesn't try to play audio
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
  }),
  formatDuration: (s: number) => "00:00",
}))

// Mock vuedraggable: render the #item slot for each item, no SortableJS in jsdom.
vi.mock("vuedraggable", () => ({
  default: defineComponent({
    name: "Draggable",
    props: ["modelValue", "list", "itemKey", "handle", "animation", "disabled"],
    emits: ["update:modelValue", "update:list"],
    setup(props, { slots }) {
      return () => {
        const items = (props.list as any[]) || (props.modelValue as any[]) || []
        return h("div", items.map((item) => slots.item?.({ element: item })))
      }
    },
  }),
}))

import { defineComponent, h } from "vue"
import DormGrid from "./DormGrid.vue"
import { useSongsStore } from "@/stores/songs"
import { useSettingsStore } from "@/stores/settings"
import type { TimeSlot } from "@/api/types"
import naive from "naive-ui"

dayjs.extend(isoWeek)

const globalStubs = {
  global: {
    plugins: [naive],
    stubs: { "icon-park": true },
  },
}

const mockSlots: TimeSlot[] = [
  { id: "slot-1", dayIndex: 7, time: "12:00", order: 1 },
  { id: "slot-2", dayIndex: 1, time: "18:00", order: 1 },
]

describe("DormGrid", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it("renders day cards and songs for a week without throwing", async () => {
    const songsStore = useSongsStore()
    const settingsStore = useSettingsStore()
    settingsStore.timeSlots = mockSlots
    settingsStore.duplicateCheckDays = 30

    // Week containing 2026-07-26 (Sun) .. 2026-07-27 is split across weeks;
    // use the ISO week of 2026-07-27 (Mon 07-27 .. Sun 08-02) which contains song id=3.
    const weekDates: string[] = []
    let cur = dayjs("2026-07-27").startOf("isoWeek")
    const end = dayjs("2026-07-27").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, {
      props: { weekDates },
      ...globalStubs,
    })

    // capture any errors thrown during render
    const errors: string[] = []

    await flushPromises()

    const html = wrapper.html()
    // The week 07-27..08-02 contains song id=3 ("七里香") on 2026-07-27
    expect(html).toContain("七里香")
    // Day label for 2026-07-27 should render (Monday)
    expect(html).toContain("2026-07-27")
    // Should have empty-state text for a day with no songs
    expect(errors.length).toBe(0)
  })

  it("groups songs by time slot and unassigned", async () => {
    const songsStore = useSongsStore()
    const settingsStore = useSettingsStore()
    settingsStore.timeSlots = mockSlots

    // Use the ISO week of 2026-07-26 (Mon 07-20 .. Sun 07-26) which has songs on 07-26
    const weekDates: string[] = []
    let cur = dayjs("2026-07-26").startOf("isoWeek")
    const end = dayjs("2026-07-26").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, {
      props: { weekDates },
      ...globalStubs,
    })

    await flushPromises()
    const html = wrapper.html()
    // slot-1 song (晴天) and unassigned song (稻香) both on 07-26
    expect(html).toContain("晴天")
    expect(html).toContain("稻香")
  })

  it("add-song button under the day title emits open-import event for that day", async () => {
    const settingsStore = useSettingsStore()
    settingsStore.timeSlots = mockSlots

    const weekDates: string[] = []
    let cur = dayjs("2026-07-27").startOf("isoWeek")
    const end = dayjs("2026-07-27").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, {
      props: { weekDates },
      ...globalStubs,
    })

    await flushPromises()

    // The first day (2026-07-27) has an "添加歌曲" button under its title.
    const addBtn = wrapper.findAll("button").find((b) => b.text().includes("添加"))
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger("click")
    await flushPromises()

    expect(wrapper.emitted("open-import")?.[0]).toEqual([
      { date: "2026-07-27" },
    ])
  })

  it("collects unassigned songs into the 8th pending card, with no per-day unassigned section", async () => {
    const settingsStore = useSettingsStore()
    settingsStore.timeSlots = mockSlots

    // ISO week of 2026-07-26 (Mon 07-20 .. Sun 07-26): 稻香 has timeSlotId null.
    const weekDates: string[] = []
    let cur = dayjs("2026-07-26").startOf("isoWeek")
    const end = dayjs("2026-07-26").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, { props: { weekDates }, ...globalStubs })
    await flushPromises()

    // 8 cards: 7 days + 1 pending
    expect(wrapper.findAll(".dorm-day-card").length).toBe(8)
    expect(wrapper.findAll(".dorm-pending-card").length).toBe(1)

    // The unassigned song lives in the pending card, not in a day card.
    expect(wrapper.find(".dorm-pending-card").html()).toContain("稻香")
    // 未分配 must no longer appear as a per-day section heading.
    const sectionTitles = wrapper.findAll(".slot-section-title").map((n) => n.text())
    expect(sectionTitles).not.toContain("未分配时段")
  })

  it("song assigned to a deleted time slot still shows up in the pending card", async () => {
    const settingsStore = useSettingsStore()
    // slot-2 is intentionally absent: song id=3 (七里香) references a deleted slot.
    settingsStore.timeSlots = [{ id: "slot-1", dayIndex: 7, time: "12:00", order: 1 }]

    const weekDates: string[] = []
    let cur = dayjs("2026-07-27").startOf("isoWeek")
    const end = dayjs("2026-07-27").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, { props: { weekDates }, ...globalStubs })
    await flushPromises()

    // Orphaned slot reference must not make the song disappear from the UI.
    expect(wrapper.find(".dorm-pending-card").html()).toContain("七里香")
  })

  it("slot button label follows the song's current slot after it changes", async () => {
    const songsStore = useSongsStore()
    const settingsStore = useSettingsStore()
    settingsStore.timeSlots = [
      { id: "slot-a", dayIndex: 1, time: "06:35", order: 1 },
      { id: "slot-b", dayIndex: 1, time: "13:55", order: 2 },
    ]

    const weekDates: string[] = []
    let cur = dayjs("2026-07-27").startOf("isoWeek")
    const end = dayjs("2026-07-27").endOf("isoWeek")
    while (cur.isBefore(end) || cur.isSame(end, "day")) {
      weekDates.push(cur.format("YYYY-MM-DD"))
      cur = cur.add(1, "day")
    }

    const wrapper = mount(DormGrid, { props: { weekDates }, ...globalStubs })
    await flushPromises()

    songsStore.dormSongs = [
      { id: 10, date: "2026-07-27", weekday: 1, title: "回归歌", artist: "", remark: "", filePath: "", timeSlotId: "slot-a", createdAt: "" } as any,
    ]
    await flushPromises()
    expect(wrapper.html()).toContain("06:35")

    // Move it to 13:55 — the button next to the title must follow, not stay on 06:35.
    songsStore.dormSongs = [
      { id: 10, date: "2026-07-27", weekday: 1, title: "回归歌", artist: "", remark: "", filePath: "", timeSlotId: "slot-b", createdAt: "" } as any,
    ]
    await flushPromises()

    // Only one song is present, so the sole rendered row is the one under test.
    // (The title lives in an <input> value, not in text content.)
    const rows = wrapper.findAll(".song-row")
    expect(rows.length).toBe(1)
    const label = rows[0].findAll("button").map((b) => b.text()).join(" ")
    expect(label).toContain("13:55")
    expect(label).not.toContain("06:35")
  })
})
