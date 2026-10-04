<template>
  <div v-if="backups.length" style="position:fixed;top:8px;right:16px;z-index:9999">
    <n-button type="warning" size="small" @click="show = true">发现未保存草稿</n-button>
  </div>
  <n-modal v-model:show="show" preset="card" title="恢复未保存歌名" style="width:min(620px,95vw)">
    <n-text>草稿将先与当前歌单比对。另一页面仍在编辑的草稿暂不列入。</n-text>
    <div v-for="backup in backups" :key="backup.key" style="margin-top:16px">
      <n-text depth="3">{{ backup.savedAt }}</n-text>
      <div v-for="draft in backup.drafts" :key="draft.id" style="padding:10px 0;border-bottom:1px solid #ddd">
        <div>{{ draft.context?.date }} · {{ draft.context?.type === 'dorm' ? '宿舍' : '播音' }} · {{ draft.title }}</div>
        <n-space style="margin-top:6px">
          <n-button size="small" :disabled="busy || !draft.context" @click="restore(backup, draft)">比对并恢复</n-button>
          <n-button size="small" :disabled="busy" @click="discard(backup, draft)">丢弃这项</n-button>
        </n-space>
        <n-text v-if="!draft.context" depth="3">旧格式草稿无法核对歌曲身份，请复制歌名后手动补回。</n-text>
      </div>
    </div>
  </n-modal>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from "vue"
import { useDialog, useMessage } from "naive-ui"
import apiClient from "../../api/client"
import type { Song } from "../../api/types"
import { databaseID, recoverableBackups, discardBackup, flushAllSaves, type DraftBackup, type StoredDraft } from "../../utils/pendingSaves"
import { writeDraftBackup } from "../../utils/persist"

const props = defineProps<{ activeClients: string[] }>()
const dialog = useDialog(); const message = useMessage()
const backups = ref<DraftBackup[]>([]); const show = ref(false); const busy = ref(false)
function load() { backups.value = recoverableBackups().filter(b => !props.activeClients.includes(b.clientId)) }
watch([databaseID, () => props.activeClients], load, { immediate: true })
const timer = setInterval(load, 5000)
onUnmounted(() => clearInterval(timer))

function confirm(content: string): Promise<boolean> {
  return new Promise(resolve => { dialog.warning({ title: '确认草稿处理', content, positiveText: '继续', negativeText: '取消', onPositiveClick: () => { resolve(true) }, onNegativeClick: () => { resolve(false) }, onClose: () => { resolve(false) }, onMaskClick: () => { resolve(false) } }) })
}
async function finish(backup: DraftBackup, draft: StoredDraft) {
  const remaining = backup.drafts.filter(d => d.id !== draft.id)
  if (remaining.length) writeDraftBackup(backup.key, { ...backup, drafts: remaining })
  else {
    discardBackup(backup.key)
    await fetch('/api/lifecycle/client', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: backup.clientId, leave: true, dirty: false }) })
  }
  load()
  if (!backups.value.length) show.value = false
}
async function restore(backup: DraftBackup, draft: StoredDraft) {
  if (!draft.context) return
  busy.value = true
  try {
    await flushAllSaves()
    const current = (await apiClient.get<Song>(`/songs/${draft.id}`)).data
    if (current.date !== draft.context.date || current.createdAt !== draft.context.createdAt) throw new Error('歌曲身份已变化，请手动核对；草稿仍保留')
    if (current.title !== draft.title) {
      if (current.title !== draft.context.originalTitle && !await confirm(`当前歌名：${current.title}。草稿：${draft.title}。当前记录已被修改，是否用草稿替换？`)) return
      await apiClient.put(`/songs/${draft.id}`, { title: draft.title, expectedTitle: current.title })
    }
    await finish(backup, draft)
    message.success('草稿已保存；重新进入歌单可看到更新')
  } catch (error: any) { message.error(error?.response?.data?.error?.message || error.message || '恢复失败，草稿保留') }
  finally { busy.value = false }
}
async function discard(backup: DraftBackup, draft: StoredDraft) {
  if (await confirm(`丢弃未保存歌名“${draft.title}”？`)) {
    try { await finish(backup, draft) } catch (error: any) { message.error(error.message) }
  }
}
</script>
