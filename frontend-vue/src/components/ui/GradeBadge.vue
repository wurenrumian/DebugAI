<template>
  <AppBadge :tone="meta.tone" :icon="meta.icon" :size="size">
    <span class="sr-only">等级：</span>{{ meta.label }}
  </AppBadge>
</template>

<script setup>
import { computed } from 'vue'
import AppBadge from './AppBadge.vue'
import { gradeMeta } from './grades'

const props = defineProps({
  grade: { type: String, default: '' },
  size: { type: String, default: 'lg' }
})

const meta = computed(() => gradeMeta(props.grade))
</script>
