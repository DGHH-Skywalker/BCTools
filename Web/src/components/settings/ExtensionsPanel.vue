<template>
  <n-space v-if="isLocal" vertical class="extensions-panel">
    <n-alert v-if="error" type="error">{{ error }}</n-alert>
    <n-card title="歌曲保留与恢复">
      <n-space align="end">
        <n-form-item label="音频保留周数"><n-input-number v-model:value="audioWeeks" :min="1" :max="520" /></n-form-item>
        <n-form-item label="歌曲信息保留年数"><n-input-number v-model:value="metadataYears" :min="1" :max="50" /></n-form-item>
        <n-button type="primary" :disabled="busy" @click="saveRetention">保存期限</n-button>
        <n-button :disabled="busy" @click="collect">立即清理</n-button>
      </n-space>
      <n-text depth="3">过期内容可在七天内从回收区恢复。</n-text>
      <n-list v-if="recoveries.length">
        <n-list-item v-for="entry in recoveries" :key="entry.id">
          <n-space justify="space-between">
            <span>{{ new Date(entry.createdAt).toLocaleString() }} · {{ entry.songCount }} 条记录 · {{ entry.fileCount }} 个文件</span>
            <n-button size="small" :disabled="busy || entry.restored" @click="restore(entry.id)">{{ entry.restored ? "已恢复" : "恢复" }}</n-button>
          </n-space>
        </n-list-item>
      </n-list>
    </n-card>
    <n-card v-if="kugouDatabase?.canSelect" title="酷狗自动解密">
      <n-space vertical>
        <n-text>
          {{ kugouDatabase.custom ? "使用选定的数据库" : "启动时自动查找本机酷狗数据库" }}
          <template v-if="kugouDatabase.keyCount > 0">，已加载 {{ kugouDatabase.keyCount }} 个密钥</template>。
        </n-text>
        <n-space>
          <n-button :disabled="busy" @click="changeKugouSource('POST')">选择本机数据库</n-button>
          <n-button v-if="kugouDatabase.custom" :disabled="busy" @click="changeKugouSource('DELETE')">恢复自动查找</n-button>
        </n-space>
      </n-space>
    </n-card>
    <n-card title="插件">
      <n-space>
        <n-button :disabled="busy" @click="installExample">安装示例布局</n-button>
        <n-button :disabled="busy" @click="fileInput?.click()">安装插件包</n-button>
        <input ref="fileInput" type="file" accept=".zip,application/zip" hidden @change="installFile" />
      </n-space>
      <n-list v-if="activePlugins.plugins.length">
        <n-list-item v-for="plugin in activePlugins.plugins" :key="plugin.manifest.id">
          <n-space vertical>
            <n-text strong>{{ plugin.manifest.name }} {{ plugin.manifest.version }}</n-text>
            <n-text depth="3">{{ plugin.manifest.permissions.join(" · ") }}</n-text>
            <n-space>
              <n-button size="small" :disabled="busy" @click="configure(plugin.manifest.id, !plugin.enabled, false)">{{ plugin.enabled ? "停用" : "启用" }}</n-button>
              <n-button v-if="plugin.manifest.layout" size="small" :disabled="busy" @click="toggleLayout(plugin.manifest.id)">
                {{ activePlugins.layoutProvider === plugin.manifest.id ? "恢复默认布局" : "使用此布局" }}
              </n-button>
              <n-button size="small" :disabled="busy" @click="removePlugin(plugin.manifest.id)">卸载</n-button>
              <n-button v-for="command in plugin.enabled ? plugin.manifest.commands : []" :key="command.id" size="small" :disabled="busy"
                @click="runCommand(plugin.manifest.id, command.id)">{{ command.label }}</n-button>
            </n-space>
          </n-space>
        </n-list-item>
      </n-list>
    </n-card>
    <n-card title="更新">
      <n-space vertical>
        <n-text v-if="!updates?.configured" depth="3">更新仓库与签名根尚未配置。</n-text>
        <n-text v-else-if="updates.source" depth="3">最近可用的更新源 {{ updates.source }}</n-text>
        <n-space>
          <n-button :disabled="busy || !updates?.configured" @click="check">检查更新</n-button>
          <n-button v-if="updates?.web" :disabled="busy" @click="rollback">恢复上一网页版本</n-button>
          <n-button v-if="updates?.coreReady" type="primary" :disabled="busy" @click="restart">安装并重启</n-button>
        </n-space>
        <n-list v-if="updates?.available.length">
          <n-list-item v-for="item in updates.available" :key="item.id">
            <n-space justify="space-between">
              <span>{{ item.description || item.id }} · {{ item.version }}</span>
              <n-button size="small" :disabled="busy" @click="apply(item.id)">{{ item.kind === "core" ? "下载并校验" : "应用" }}</n-button>
            </n-space>
          </n-list-item>
        </n-list>
      </n-space>
    </n-card>
  </n-space>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue"
import { useMessage } from "naive-ui"
import { activePlugins, apiV1, checkWebRevision, localCall, refreshExtensions } from "../../api/extensions"
import type { PluginSnapshot, UpdateStatus } from "../../api/extensions"
import { runsInDesktop } from "../../utils/desktop"

import type { components } from "../../api/generated/contracts"

type Retention = components["schemas"]["RetentionStatus"]
type Recovery = components["schemas"]["Recovery"]
type KugouDatabase = components["schemas"]["KugouDatabaseStatus"]

const isLocal = runsInDesktop() || ["localhost", "127.0.0.1", "[::1]"].includes(location.hostname)
const audioWeeks = ref<number | null>(4)
const metadataYears = ref<number | null>(2)
const recoveries = ref<Recovery[]>([])
const updates = ref<UpdateStatus>()
const kugouDatabase = ref<KugouDatabase>()
const fileInput = ref<HTMLInputElement>()
const busy = ref(false)
const error = ref("")
const message = useMessage()

function showRetention(data: Retention) {
  audioWeeks.value = data.settings.audioWeeks
  metadataYears.value = data.settings.metadataYears
  recoveries.value = data.recoveries
}

async function perform(work: () => Promise<unknown>) {
  busy.value = true
  error.value = ""
  try {
    await work()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
    message.error(error.value)
  } finally {
    busy.value = false
  }
}

const saveRetention = () => perform(async () => showRetention(await localCall<Retention>("/retention", "PUT", JSON.stringify({
  audioWeeks: audioWeeks.value, metadataYears: metadataYears.value,
}))))
const collect = () => perform(async () => showRetention(await localCall<Retention>("/retention/collect", "POST")))
const restore = (id: string) => perform(async () => showRetention(await localCall<Retention>("/retention/recoveries/" + id + "/restore", "POST")))
const changeKugouSource = (method: "POST" | "DELETE") => perform(async () => {
  const status = await localCall<KugouDatabase | undefined>("/decrypt/kugou/source", method)
  if (status) kugouDatabase.value = status
})
const installExample = () => perform(async () => { activePlugins.value = await localCall<PluginSnapshot>("/plugins/example", "POST") })
const configure = (id: string, enabled: boolean, layout: boolean) => perform(async () => {
  activePlugins.value = await localCall<PluginSnapshot>("/plugins/" + id, "PUT", JSON.stringify({ enabled, layout }))
})
const toggleLayout = (id: string) => configure(id, true, activePlugins.value.layoutProvider !== id)
const removePlugin = (id: string) => perform(async () => { activePlugins.value = await localCall<PluginSnapshot>("/plugins/" + id, "DELETE") })
const runCommand = (id: string, command: string) => perform(async () => {
  const result = await localCall<{ output: string }>("/plugins/" + id + "/commands/" + command, "POST", "{}")
  message.success(result.output || "命令已完成")
})

function installFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ""
  if (file) void perform(async () => { activePlugins.value = await localCall<PluginSnapshot>("/plugins", "POST", file) })
}

const check = () => perform(async () => { updates.value = await localCall<UpdateStatus>("/updates/check", "POST") })
const apply = (id: string) => perform(async () => {
  updates.value = await localCall<UpdateStatus>("/updates/" + id + "/apply", "POST")
  await refreshExtensions()
  await checkWebRevision()
})
const rollback = () => perform(async () => {
  updates.value = await localCall<UpdateStatus>("/updates/web/rollback", "POST")
  await checkWebRevision()
})
const restart = () => perform(() => localCall("/updates/restart", "POST"))

onMounted(() => {
  if (isLocal) void localCall<KugouDatabase>("/decrypt/kugou/status")
    .then(status => { kugouDatabase.value = status }).catch(() => {})
  if (isLocal) void perform(async () => {
    const [retention, status] = await Promise.all([
      localCall<Retention>("/retention"), apiV1<UpdateStatus>("/updates"), refreshExtensions(),
    ])
    showRetention(retention)
    updates.value = status
  })
})
</script>

<style scoped>
.extensions-panel { margin-top: 16px; }
</style>
