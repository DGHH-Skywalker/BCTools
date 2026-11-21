<template>
  <PageContainer>
    <n-card v-if="page" :title="page.title"><n-p>{{ page.text }}</n-p></n-card>
    <n-empty v-else description="此插件页面当前不可用" />
  </PageContainer>
</template>

<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import { activePlugins } from "../api/extensions"
import PageContainer from "../components/ui/PageContainer.vue"

const route = useRoute()
const page = computed(() => activePlugins.value.plugins
  .find(item => item.enabled && item.manifest.id === route.params.plugin)?.manifest.pages
  ?.find(item => item.id === route.params.page))
</script>
