<template>
  <n-button
    circle
    type="primary"
    style="position:fixed;right:24px;bottom:24px;z-index:100;"
    @click="$emit('click')"
  >
    <template #icon>
      <slot name="icon" />
    </template>
  </n-button>
</template>

<script setup lang="ts">
defineEmits<{
  (e: "click"): void
}>()
</script>
