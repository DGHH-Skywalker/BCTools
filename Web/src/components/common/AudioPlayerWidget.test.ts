import { describe, it, expect, vi, beforeEach } from "vitest"
import { mount } from "@vue/test-utils"
import { nextTick } from "vue"

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

// useAudioPlayer 在模块顶层就 `new Audio()`，所以必须在 import 它之前把 Audio 换掉。
// vi.stubGlobal 在 import 之后才生效，这里改用直接赋值 + vi.hoisted。
const { FakeAudioCtor } = vi.hoisted(() => {
  class FakeAudio {
    paused = true
    src = ""
    currentTime = 0
    duration = 0
    preload = ""
    error: any = null
    addEventListener() {}
    removeEventListener() {}
    load() {}
    play() {
      this.paused = false
      return Promise.resolve()
    }
    pause() {
      this.paused = true
    }
  }
  ;(globalThis as any).Audio = FakeAudio
  ;(globalThis as any).MediaMetadata = class {}
  return { FakeAudioCtor: FakeAudio }
})
void FakeAudioCtor

import AudioPlayerWidget from "@/components/common/AudioPlayerWidget.vue"
import { useAudioPlayer } from "@/composables/useAudioPlayer"

// 必须用真实 DOMRect：useAudioPlayer.show() 靠 `source instanceof DOMRect`
// 区分「从按钮展开」和「指定坐标」，鸭子类型的假对象会走错分支。
function rect(): DOMRect {
  return new DOMRect(24, 500, 40, 40)
}

// Naive UI 组件在单测里不自动解析（靠 unplugin-vue-components），全部打桩。
const STUBS = {
  NCard: { template: "<div><slot name='header'/><slot/></div>" },
  NSpace: { template: "<div><slot/></div>" },
  NText: { template: "<span><slot/></span>" },
  NEllipsis: { template: "<span><slot/></span>" },
  NSlider: { template: "<div/>" },
  NButton: { template: "<button><slot/></button>" },
  Close: { template: "<i/>" },
  Play: { template: "<i/>" },
  Pause: { template: "<i/>" },
}

describe("AudioPlayerWidget open → close → open", () => {
  let player: ReturnType<typeof useAudioPlayer>

  beforeEach(() => {
    player = useAudioPlayer()
    player.stop()
    player.close()
  })

  it("is visible again after being closed and reopened", async () => {
    // 组件在 App.vue 里是常挂载的（v-show），只有内部 wrapper 用 v-if。
    const wrapper = mount(AudioPlayerWidget, {
      attachTo: document.body,
      global: { stubs: STUBS },
    })

    // 1) 从按钮位置展开
    player.play("/songs/a.mp3", "测试歌曲")
    player.show(rect())
    await nextTick()
    let el = wrapper.find(".audio-player-wrapper")
    expect(el.exists()).toBe(true)

    // 2) 关闭。走组件内部的 closeAnimated（带 260ms 收起动画），
    // 这正是用户点右上角 × 的路径。
    ;(wrapper.vm as any).closeAnimated()
    await sleep(320)
    await nextTick()
    expect(wrapper.find(".audio-player-wrapper").exists()).toBe(false)

    // 3) 再次打开：必须真的可见。
    // 回归点：收起动画把 transform 留在 scale(0.2)/opacity:0，而 animateOpen 只在
    // onMounted 跑过一次，导致第二次打开时组件渲染出来却完全透明——
    // 用户看到的现象是「按钮一点就没了，播放器没出现」。
    player.show(rect())
    await nextTick()
    el = wrapper.find(".audio-player-wrapper")
    expect(el.exists()).toBe(true)

    // 等进入动画跑完
    await sleep(420)
    await nextTick()

    const style = el.attributes("style") || ""
    expect(style).not.toMatch(/opacity:\s*0\s*[;"]/)
    expect(style).not.toMatch(/scale\(0\.2\)/)

    wrapper.unmount()
  })

  it("reopening without an origin rect is fully opaque", async () => {
    const wrapper = mount(AudioPlayerWidget, {
      attachTo: document.body,
      global: { stubs: STUBS },
    })

    player.play("/songs/a.mp3", "测试歌曲")
    player.show(rect())
    await nextTick()
    ;(wrapper.vm as any).closeAnimated()
    await sleep(320)
    await nextTick()

    // 无 originRect 的展开（例如通过 toggle）也不能残留透明状态
    player.show()
    await nextTick()
    await sleep(420)
    await nextTick()

    const style = wrapper.find(".audio-player-wrapper").attributes("style") || ""
    expect(style).not.toMatch(/opacity:\s*0\s*[;"]/)

    wrapper.unmount()
  })
})
