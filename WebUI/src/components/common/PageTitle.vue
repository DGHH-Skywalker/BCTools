<template>
  <div class="page-header">
    <div class="title-row">
      <div v-if="$slots.prefix" class="title-prefix">
        <slot name="prefix" />
      </div>
      <n-h1 class="page-title">{{ title }}</n-h1>
      <div v-if="$slots.extra" class="title-extra">
        <slot name="extra" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NH1 } from "naive-ui"

defineProps<{
  title: string
}>()
</script>

<style scoped>
.page-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px 16px 8px;
}
.page-title {
  margin: 16px 0 0;
  font-family: "JiangxiZhuokai", var(--font-title), "Microsoft YaHei", "PingFang SC", sans-serif;
  color: var(--theme-color);
  font-size: 2rem;
  line-height: 1.2;
  text-align: center;
}
.title-row {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
}
.title-extra {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  color: var(--theme-color);
}
.title-prefix {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
}
@media (max-width: 768px) {
  .page-title {
    font-size: 1.5rem;
  }
}
</style>
