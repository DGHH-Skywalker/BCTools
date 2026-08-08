<template>
  <span class="warning-text">
    <slot />
  </span>
</template>

<style scoped>
.warning-text {
  color: var(--color-warning);
  font-size: var(--spacing-md);
  line-height: 1.4;
}
</style>
