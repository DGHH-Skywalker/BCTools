<template>
  <div>
    <PageTitle :title="t('dormManage.title')">
      <template #extra>
        <n-button quaternary circle @click="showSlotModal = true">
          <template #icon>
            <Time theme="outline" :size="22" :strokeWidth="3" />
          </template>
        </n-button>
      </template>
    </PageTitle>
    <SongTable type="dorm" />
    <TimeSlotModal v-model:show="showSlotModal" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import SongTable from "../components/common/SongTable.vue"
import TimeSlotModal from "../components/dorm/TimeSlotModal.vue"
import PageTitle from "../components/common/PageTitle.vue"
import { Time } from "@icon-park/vue-next"

const { t } = useI18n()
const settingsStore = useSettingsStore()
const showSlotModal = ref(false)

onMounted(() => {
  settingsStore.fetchSettings()
})
</script>

