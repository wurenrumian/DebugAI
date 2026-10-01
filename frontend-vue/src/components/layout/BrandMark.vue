<template>
  <component :is="to ? RouterLink : 'span'" :to="to || undefined" class="brand">
    <span class="brand-logo" aria-hidden="true">
      <AppIcon name="code" :size="18" :stroke-width="2.2" />
    </span>
    <span class="brand-text">
      <span class="brand-name">AI 教学辅助平台</span>
      <span v-if="tagline" class="brand-tagline">C / C++ 程序设计</span>
    </span>
  </component>
</template>

<script setup>
import { RouterLink } from 'vue-router'
import AppIcon from '../ui/AppIcon.vue'

defineProps({
  to: { type: String, default: '' },
  tagline: { type: Boolean, default: false }
})
</script>

<style scoped>
.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--color-text);
  text-decoration: none;
}

.brand:hover {
  text-decoration: none;
}

.brand-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: var(--radius-md);
  color: var(--color-on-primary);
  background: var(--color-primary);
}

.brand-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.2;
}

.brand-name {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  white-space: nowrap;
}

.brand-tagline {
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}
</style>
