<template>
  <n-card title="数据目录与旧歌单">
    <n-space vertical>
      <n-text style="overflow-wrap:anywhere">固定目录：{{ directory || '正在读取…' }}</n-text>
      <n-text depth="3">程序移动或升级后继续使用此目录。导入旧歌单前自动备份，选择的记录会新增到当前歌单。</n-text>
      <n-space><n-button size="small" @click="load">刷新来源</n-button><n-button size="small" @click="addDirectory">添加程序或旧数据目录</n-button></n-space>
      <div v-for="source in sources" :key="source.id" style="border-top:1px solid #ddd;padding-top:10px">
        <div style="overflow-wrap:anywhere">{{ source.directory }}</div>
        <n-text v-if="source.error" type="error">{{ source.error }}</n-text>
        <template v-else>
          <div>宿舍 {{ source.dormCount }} 首 · 播音 {{ source.broadcastCount }} 首 · {{ source.firstDate }} 至 {{ source.lastDate }}</div>
          <n-text v-if="source.missingFiles" type="warning">缺少音频 {{ source.missingFiles }} 项，请先核对来源文件</n-text>
          <n-button size="small" @click="preview(source)">比对并选择记录</n-button>
        </template>
      </div>
      <n-text v-if="!sources.length" depth="3">尚未发现其他数据来源。</n-text>
    </n-space>
  </n-card>
  <n-modal v-model:show="show" preset="card" title="补回旧歌单" style="width:min(700px,95vw)">
    <n-text>当前歌单会先备份。不同记录会使用新 ID 导入，当前记录保留。</n-text>
    <n-checkbox-group v-model:value="selected">
      <n-space vertical style="max-height:55vh;overflow:auto;margin:12px 0">
        <n-checkbox v-for="record in records" :key="key(record)" :value="key(record)" :disabled="record.identical">
          {{ record.song.date }} · {{ record.type === 'dorm' ? '宿舍' : '播音' }} · {{ record.song.title }}
          <span v-if="record.identical">（相同，已存在）</span>
          <span v-else-if="record.existing">（ID 冲突，现有：{{ record.existing.title }}；将另存一条）</span>
        </n-checkbox>
      </n-space>
    </n-checkbox-group>
    <n-button type="primary" :loading="busy" :disabled="!selected.length" @click="importSelected">备份并导入所选 {{ selected.length }} 条</n-button>
  </n-modal>
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useDialog, useMessage } from "naive-ui"
import apiClient from "../../api/client"
import type { Song, SongType } from "../../api/types"
import { flushAllSaves } from "../../utils/pendingSaves"

type Source = { id: string; directory: string; digest: string; dormCount: number; broadcastCount: number; firstDate: string; lastDate: string; missingFiles: number; error?: string }
type RecordRow = { type: SongType; song: Song; existing?: Song; identical: boolean }
const directory = ref(''); const sources = ref<Source[]>([]); const records = ref<RecordRow[]>([])
const current = ref<Source>(); const selected = ref<string[]>([]); const show = ref(false); const busy = ref(false)
const message = useMessage(); const dialog = useDialog()
const key = (record: RecordRow) => `${record.type}:${record.song.id}`
async function load() {
  try { const { data } = await apiClient.get('/system/data-sources'); directory.value = data.directory; sources.value = data.sources }
  catch (error: any) { message.error(error?.response?.data?.error?.message || '数据来源只可在运行程序的电脑上查看') }
}
async function addDirectory() {
  try {
    const { data } = await apiClient.post('/files/select-dir', {})
    const path = data.path || data.directory
    if (!path) return
    await apiClient.post('/system/search-directory', { directory: path })
    await load(); message.success('目录已登记，下次启动会检查其中的程序版本')
  } catch (error: any) { message.error(error?.response?.data?.error?.message || error.message || '目录添加失败') }
}
async function preview(source: Source) {
  try {
    await flushAllSaves()
    const { data } = await apiClient.get(`/system/data-sources/${source.id}`)
    current.value = data.source; records.value = data.records; selected.value = []; show.value = true
  } catch (error: any) { message.error(error?.response?.data?.error?.message || '比对失败') }
}
function importSelected() {
  dialog.warning({ title: '确认补回旧记录', content: `将导入 ${selected.value.length} 条所选记录；发生 ID 冲突的记录会新增，原记录保留。`, positiveText: '备份并导入', negativeText: '取消', onPositiveClick: performImport })
}
async function performImport() {
  if (!current.value) return
  busy.value = true
  try {
    await flushAllSaves()
    const { data } = await apiClient.post('/system/data-sources/import', {
      sourceId: current.value.id, digest: current.value.digest, confirmed: true,
      records: records.value.filter(record => selected.value.includes(key(record))).map(record => ({ type: record.type, id: record.song.id })),
    })
    message.success(`已补回 ${data.imported} 条记录，重新进入歌单可查看`); show.value = false; await load()
  } catch (error: any) { message.error(error?.response?.data?.error?.message || '导入失败，原来源保留') }
  finally { busy.value = false }
}
onMounted(load)
</script>
