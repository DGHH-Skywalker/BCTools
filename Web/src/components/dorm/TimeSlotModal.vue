<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="t('settings.advancedSettings')"
    :style="{ width: '90%', maxWidth: '800px' }"
    :content-style="{ padding: '16px' }"
    @update:show="emit('update:show', $event)"
  >
    <n-space vertical size="large" style="width:100%;">
      <n-card size="small" :title="t('timeSlots.duplicateCheckSettings')">
        <n-space align="center">
          <n-text>{{ t("settings.duplicateCheckDays") }}:</n-text>
          <n-input-number v-model:value="duplicateCheckDaysModel" :min="1" :max="365" style="width:120px" />
          <n-text depth="3">{{ t("settings.duplicateCheckDaysHint") }}</n-text>
        </n-space>
      </n-card>

      <n-card size="small" :title="t('timeSlots.title')">
        <n-space vertical style="width:100%;">
          <n-select
            v-model:value="selectedDay"
            :options="dayOptions"
            :placeholder="t('timeSlots.selectDaysPlaceholder')"
          />

          <div v-if="selectedDay" style="display:flex;flex-direction:column;gap:16px;margin-top:8px;">
            <n-card size="small" :title="weekdayLabel(selectedDay)">
              <n-data-table :columns="slotColumns" :data="daySlots" :bordered="true" :max-height="240" />
              <n-button dashed style="margin-top:8px;" @click="addSlot">
                + {{ t("timeSlots.addSlot") }}
              </n-button>
            </n-card>
          </div>

          <n-empty v-else :description="t('timeSlots.selectDaysPlaceholder')" />
        </n-space>
      </n-card>
    </n-space>

    <template #footer>
      <n-space justify="end">
        <n-button @click="close">{{ t("common.cancel") }}</n-button>
        <n-button type="primary" :loading="saving" @click="save">{{ t("common.save") }}</n-button>
      </n-space>
    </template>

    <n-modal
      v-model:show="showConfirm"
      preset="dialog"
      :title="t('common.confirm')"
      :content="confirmContent"
      positive-text="确认"
      negative-text="取消"
      @positive-click="confirmSave"
    />
  </n-modal>
</template>

<script setup lang="ts">
import { ref, watch, h, computed } from "vue"
import { useI18n } from "../../i18n"
import { useSettingsStore } from "../../stores/settings"
import type { TimeSlot } from "../../api/types"
import { NButton, NInputNumber, NTimePicker, useMessage } from "naive-ui"
import type { SelectOption } from "naive-ui"

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
}>()

const { t, weekdayLabel } = useI18n()
const settingsStore = useSettingsStore()
const message = useMessage()

const localSlots = ref<TimeSlot[]>([])
const duplicateCheckDaysModel = ref(30)
const selectedDay = ref<number | null>(null)
const saving = ref(false)
const showConfirm = ref(false)
const confirmContent = ref("")
const pendingSlotsRef = ref<TimeSlot[] | null>(null)

const dayOptions = computed<SelectOption[]>(() =>
  Array.from({ length: 7 }, (_, i) => ({ label: weekdayLabel(i + 1), value: i + 1 }))
)

watch(
  () => props.show,
  (show) => {
    if (show) {
      localSlots.value = settingsStore.timeSlots.map((s) => ({ ...s }))
      duplicateCheckDaysModel.value = settingsStore.duplicateCheckDays || 30
      const today = new Date().getDay()
      selectedDay.value = today === 0 ? 7 : today
    }
  },
  { immediate: true },
)

const daySlots = computed(() => {
  if (!selectedDay.value) return []
  return localSlots.value
    .filter((s) => s.dayIndex === selectedDay.value)
    .sort((a, b) => a.order - b.order)
})

function formatTime(ms: number): string {
  const d = new Date(ms)
  const h = String(d.getHours()).padStart(2, "0")
  const m = String(d.getMinutes()).padStart(2, "0")
  return `${h}:${m}`
}

function parseTime(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return new Date(1970, 0, 1, h, m).getTime()
}

function updateSlot(id: string, patch: Partial<TimeSlot>) {
  const idx = localSlots.value.findIndex((s) => s.id === id)
  if (idx >= 0) {
    localSlots.value[idx] = { ...localSlots.value[idx], ...patch }
  }
}

const slotColumns = [
  {
    title: t("timeSlots.order"),
    key: "order",
    width: 90,
    render: (row: TimeSlot) =>
      h(NInputNumber, {
        value: row.order,
        min: 1,
        style: "width:70px",
        onUpdateValue: (v: number | null) => {
          if (v != null) updateSlot(row.id, { order: v })
        },
      }),
  },
  {
    title: t("timeSlots.time"),
    key: "time",
    width: 160,
    render: (row: TimeSlot) =>
      h(NTimePicker, {
        value: parseTime(row.time),
        format: "HH:mm",
        style: "width:120px",
        onUpdateValue: (v: number | null) => {
          if (v != null) updateSlot(row.id, { time: formatTime(v) })
        },
      }),
  },
  {
    title: "",
    key: "actions",
    width: 80,
    render: (row: TimeSlot) =>
      h(
        NButton,
        { size: "small", type: "error", onClick: () => deleteSlot(row.id) },
        () => t("timeSlots.deleteSlot"),
      ),
  },
]

function addSlot() {
  if (!selectedDay.value) return
  const dayList = localSlots.value.filter((s) => s.dayIndex === selectedDay.value)
  const tempId = crypto.randomUUID?.() || `temp-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  localSlots.value.push({
    id: tempId,
    dayIndex: selectedDay.value,
    order: dayList.length + 1,
    time: "12:00",
  })
}

function deleteSlot(id: string) {
  localSlots.value = localSlots.value.filter((s) => s.id !== id)
}

function close() {
  emit("update:show", false)
}

async function save() {
  saving.value = true
  try {
    const sorted = [...localSlots.value].sort((a, b) => {
      if (a.dayIndex !== b.dayIndex) return a.dayIndex - b.dayIndex
      return a.order - b.order
    })
    const result = await settingsStore.updateSettings({
      timeSlots: sorted,
      duplicateCheckDays: duplicateCheckDaysModel.value ?? 30,
    } as any)
    if (result?.confirmNeeded) {
      pendingSlotsRef.value = sorted
      confirmContent.value = t("timeSlots.deleteConfirm", { count: result.affectedCount })
      showConfirm.value = true
      saving.value = false
      return
    }
    message.success(t("common.save"))
    close()
  } catch (err: any) {
    message.error(err?.message || "保存失败")
  } finally {
    saving.value = false
  }
}

async function confirmSave() {
  if (!pendingSlotsRef.value) return
  saving.value = true
  try {
    await settingsStore.updateSettings({
      timeSlots: pendingSlotsRef.value,
      duplicateCheckDays: duplicateCheckDaysModel.value ?? 30,
      confirmed: true,
    } as any)
    message.success(t("common.save"))
    close()
  } catch (err: any) {
    message.error(err?.message || "保存失败")
  } finally {
    pendingSlotsRef.value = null
    saving.value = false
  }
}
</script>
