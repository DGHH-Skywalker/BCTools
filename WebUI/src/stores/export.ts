import { defineStore } from "pinia"
import { ref } from "vue"

export const useExportStore = defineStore("export", () => {
  const selectedDates = ref<string[]>([])
  const templateType = ref("simple")
  const customTemplateHTML = ref("")
  const backgroundImage = ref<string | null>(null)
  const backgroundColor = ref("#ffffff")

  function setDates(dates: string[]) {
    selectedDates.value = [...new Set(dates)].sort()
  }

  function addDate(date: string) {
    if (!selectedDates.value.includes(date)) {
      selectedDates.value.push(date)
      selectedDates.value.sort()
    }
  }

  function removeDate(date: string) {
    selectedDates.value = selectedDates.value.filter((d) => d !== date)
  }

  return { selectedDates, templateType, customTemplateHTML, backgroundImage, backgroundColor, setDates, addDate, removeDate }
})
