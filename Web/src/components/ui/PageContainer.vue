<template>
  <div class="page-container" :class="sizeClass">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  size?: "default" | "large" | "xlarge"
}>()

const sizeClass = computed(() => {
  switch (props.size) {
    case "large":
      return "page-container-lg"
    case "xlarge":
      return "page-container-xl"
    default:
      return ""
  }
})
</script>

<style scoped>
.page-container {
  padding: var(--page-padding);
  max-width: var(--page-max-width);
  margin: 0 auto;
}

.page-container-lg {
  max-width: var(--page-max-width-lg);
}

.page-container-xl {
  max-width: var(--page-max-width-xl);
}
</style>
