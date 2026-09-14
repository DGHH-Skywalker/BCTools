<template>
  <div class="dorm-timeslots">
    <n-space align="center" style="margin-bottom:16px;"><n-h2 style="margin:0;">宿舍时段与查重</n-h2></n-space>

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
          <n-tabs v-model:value="selectedDay" type="segment">
            <n-tab v-for="day in 7" :key="day" :name="day" :tab="weekdayLabel(day)" />
          </n-tabs>

          <div class="day-slots">
            <draggable
              v-model="slotsByDay[selectedDay]"
              item-key="id"
              handle=".drag-handle"
              :animation="150"
              ghost-class="slot-row-ghost"
              class="slot-draggable"
            >
              <template #item="{ element: slot }">
                <div class="slot-row">
                  <span class="drag-handle" :title="t('dorm.dragSort')">
                    <svg viewBox="0 0 10 16" width="10" height="16" aria-hidden="true">
                      <circle cx="2.5" cy="3" r="1.3" fill="currentColor" />
                      <circle cx="2.5" cy="8" r="1.3" fill="currentColor" />
                      <circle cx="2.5" cy="13" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="3" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="8" r="1.3" fill="currentColor" />
                      <circle cx="7" cy="13" r="1.3" fill="currentColor" />
                    </svg>
                  </span>
                  <n-time-picker
                    v-model:formatted-value="slot.time"
                    format="HH:mm"
                    size="small"
                    style="width:110px"
                  />
                  <n-button size="small" quaternary type="error" @click="deleteSlot(slot.id)">
                    <template #icon>
                      <Delete theme="outline" :size="14" :strokeWidth="3" />
                    </template>
                    {{ t("timeSlots.deleteSlot") }}
                  </n-button>
                </div>
              </template>
            </draggable>

            <n-empty
              v-if="slotsByDay[selectedDay].length === 0"
              size="small"
              :description="t('timeSlots.emptyDay')"
              style="padding:12px 0;"
            />

            <n-space style="margin-top:8px;">
              <n-button dashed @click="addSlot">+ {{ t("timeSlots.addSlot") }}</n-button>
              <n-button @click="applyToAllDays">{{ t("timeSlots.applyToAllDays") }}</n-button>
            </n-space>
            <n-text depth="3" style="font-size:12px;display:block;margin-top:6px;">
              {{ t("timeSlots.applyToAllDaysHint") }}
            </n-text>
          </div>
        </n-space>
      </n-card>
    </n-space>

    <div class="actions">
      <n-button type="primary" :loading="saving" @click="save">{{ t("common.save") }}</n-button>
    </div>

    <n-modal
      v-model:show="showConfirm"
      preset="dialog"
      :title="t('common.confirm')"
      :content="confirmContent"
      positive-text="确认"
      negative-text="取消"
      @positive-click="confirmSave"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue"
import { useI18n } from "../i18n"
import { useSettingsStore } from "../stores/settings"
import type { TimeSlot } from "../api/types"
import { useMessage } from "naive-ui"
import draggable from "vuedraggable"
import { Delete } from "@icon-park/vue-next"

const { t, weekdayLabel } = useI18n()
const settingsStore = useSettingsStore()
const message = useMessage()

const slotsByDay = reactive<Record<number, TimeSlot[]>>({
  1: [], 2: [], 3: [], 4: [], 5: [], 6: [], 7: [],
})
const duplicateCheckDaysModel = ref(30)
const selectedDay = ref(1)
const saving = ref(false)
const showConfirm = ref(false)
const confirmContent = ref("")

function buildFromSettings() {
  for (let day = 1; day <= 7; day++) {
    slotsByDay[day] = settingsStore.timeSlots
      .filter((s) => s.dayIndex === day)
      .sort((a, b) => a.order - b.order)
      .map((s) => ({ ...s }))
  }
  duplicateCheckDaysModel.value = settingsStore.duplicateCheckDays || 30
}

function newSlotId(): string {
  return crypto.randomUUID?.() || `temp-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

function addSlot() {
  slotsByDay[selectedDay.value].push({
    id: newSlotId(),
    dayIndex: selectedDay.value,
    order: slotsByDay[selectedDay.value].length + 1,
    time: "12:00",
  })
}

function deleteSlot(id: string) {
  slotsByDay[selectedDay.value] = slotsByDay[selectedDay.value].filter((s) => s.id !== id)
}

function applyToAllDays() {
  const source = slotsByDay[selectedDay.value].map((s) => ({ ...s }))
  if (source.length === 0) {
    message.warning(t("timeSlots.emptyDay"))
    return
  }
  for (let day = 1; day <= 7; day++) {
    if (day === selectedDay.value) continue
    slotsByDay[day] = source.map((s, i) => ({
      id: newSlotId(),
      dayIndex: day,
      order: i + 1,
      time: s.time,
    }))
  }
  message.success(t("timeSlots.applyToAllDaysDone"))
}

function flattenSlots(): TimeSlot[] {
  const all: TimeSlot[] = []
  for (let day = 1; day <= 7; day++) {
    slotsByDay[day].forEach((s, i) => {
      all.push({ ...s, dayIndex: day, order: i + 1 })
    })
  }
  return all
}

async function save() {
  await doSave(false)
}

async function confirmSave() {
  await doSave(true)
}

async function doSave(confirmed: boolean) {
  saving.value = true
  try {
    const result = await settingsStore.updateSettings({
      timeSlots: flattenSlots(),
      duplicateCheckDays: duplicateCheckDaysModel.value ?? 30,
      confirmed,
    } as any)
    if (result?.confirmNeeded) {
      confirmContent.value = t("timeSlots.deleteConfirm", { count: result.affectedCount })
      showConfirm.value = true
      return
    }
    message.success(t("common.save"))
  } catch (err: any) {
    message.error(err?.message || "保存失败")
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (!settingsStore.timeSlots.length) {
    await settingsStore.fetchSettings()
  }
  buildFromSettings()
  const today = new Date().getDay()
  selectedDay.value = today === 0 ? 7 : today
})
</script>

<style scoped>
.dorm-timeslots {
  padding: 16px;
  max-width: 720px;
  margin: 0 auto;
}

.day-slots {
  margin-top: 12px;
}

.slot-draggable {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 32px;
}

.slot-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 8px;
  border: 1px solid var(--color-border, #e0e0e6);
  border-radius: var(--radius-sm);
}

.drag-handle {
  cursor: grab;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
}

.slot-row-ghost {
  opacity: 0.5;
}

.actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
</style>
