<template>
  <button
    type="button"
    class="theme-toggle"
    :aria-label="label"
    :title="label"
    @click="toggleTheme"
  >
    <AppIcon :name="themeState.resolved === 'dark' ? 'sun' : 'moon'" :size="18" />
  </button>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'
import { themeState, toggleTheme } from '../../utils/theme'

const label = computed(() => (themeState.resolved === 'dark' ? '切换到浅色主题' : '切换到深色主题'))
</script>

<style scoped>
.theme-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  transition: background-color var(--duration-fast) var(--ease), color var(--duration-fast) var(--ease);
}

.theme-toggle:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}
</style>
