<template>
  <n-modal v-model:show="show" preset="card" :title="latest?.title" :style="{ width: 'min(560px, 92vw)' }">
    <template v-if="latest">
      <n-space vertical size="large">
        <n-text depth="3">版本 {{ latest.version }} · {{ publishedDate }}</n-text>
        <n-ul class="update-items">
          <n-li v-for="item in latest.content" :key="item">{{ item }}</n-li>
        </n-ul>
        <n-space justify="end">
          <n-button @click="show = false">知道了</n-button>
          <n-button v-if="latest.downloadUrl" tag="a" type="primary" :href="latest.downloadUrl" target="_blank" rel="noopener noreferrer">
            查看发布页
          </n-button>
        </n-space>
      </n-space>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { getLatestUpdateLog } from "../../api/misc"
import type { UpdateLog } from "../../api/types"
import { claimUpdateLog } from "../../utils/updateLog"

const latest = ref<UpdateLog | null>(null)
const show = ref(false)
const publishedDate = computed(() => latest.value ? new Date(latest.value.publishedAt).toLocaleDateString() : "")

onMounted(async () => {
  try {
    const log = await getLatestUpdateLog()
    if (!claimUpdateLog(log)) return
    latest.value = log
    show.value = true
  } catch {
    // Update notices must never block navigation or startup.
  }
})
</script>

<style scoped>
.update-items { margin: 0; padding-left: 20px; }
</style>
