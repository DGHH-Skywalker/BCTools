<template>
  <div class="home-container" @contextmenu.prevent>
    <div v-if="mode === 'home'" class="home-content" @contextmenu.prevent>
      <n-image
        src="/logo.png"
        alt="logo"
        class="home-logo"
        preview-disabled
        :img-props="{ draggable: false }"
        @contextmenu.prevent
      />
      <n-h1 class="home-title" @contextmenu.prevent>{{ t("home.title") }}</n-h1>
      <n-space>
        <n-button type="primary" size="large" @click="$router.push('/dorm/manage')">{{ t("home.startButton") }}</n-button>
        <n-button size="large" @click="mode = 'feasibility'">{{ t("home.feasibilityCheckButton") }}</n-button>
      </n-space>
    </div>

    <FeasibilityCheck v-else @back="mode = 'home'" />
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "../i18n"
import FeasibilityCheck from "../components/home/FeasibilityCheck.vue"

const { t } = useI18n()
const mode = ref<"home" | "feasibility">("home")
</script>

<style scoped>
.home-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 60px 20px;
  box-sizing: border-box;
  user-select: none;
  -webkit-user-select: none;
}

.home-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  text-align: center;
  user-select: none;
  -webkit-user-select: none;
}

.home-logo {
  width: 160px;
  height: 160px;
  object-fit: contain;
  pointer-events: none;
  -webkit-user-drag: none;
  user-select: none;
  -webkit-user-select: none;
}

.home-logo :deep(img) {
  pointer-events: none;
  -webkit-user-drag: none;
  user-select: none;
  -webkit-user-select: none;
}

.home-title {
  margin: 0;
  font-family: var(--font-title);
  color: var(--theme-color);
  font-size: 3rem;
  line-height: 1.2;
  user-select: none;
  -webkit-user-select: none;
}

@media (max-width: 768px) {
  .home-logo {
    width: 120px;
    height: 120px;
  }

  .home-title {
    font-size: 2.25rem;
  }
}

@media (max-width: 480px) {
  .home-logo {
    width: 100px;
    height: 100px;
  }

  .home-title {
    font-size: 1.75rem;
  }
}
</style>
