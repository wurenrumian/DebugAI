<template>
  <span :class="['badge', `badge-${tone}`, `badge-${size}`]">
    <AppIcon v-if="icon" :name="icon" :size="size === 'lg' ? 15 : 13" :stroke-width="2.2" />
    <slot />
  </span>
</template>

<script setup>
import AppIcon from './AppIcon.vue'

defineProps({
  // neutral | primary | success | warning | danger | info | excellent | pass | improve
  tone: { type: String, default: 'neutral' },
  // sm | md | lg
  size: { type: String, default: 'md' },
  icon: { type: String, default: '' }
})
</script>

<style scoped>
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  font-weight: var(--fw-medium);
  line-height: 1.4;
  white-space: nowrap;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
}

.badge-sm {
  padding: 0 8px;
  font-size: 11px;
}

.badge-md {
  padding: 1px 10px;
  font-size: var(--fs-xs);
}

.badge-lg {
  padding: 3px 12px;
  font-size: var(--fs-sm);
}

.badge-neutral {
  color: var(--color-text-secondary);
  background: var(--color-surface-3);
  border-color: var(--color-border);
}

.badge-primary,
.badge-info {
  color: var(--color-info);
  background: var(--color-info-soft);
  border-color: var(--color-info-border);
}

.badge-success {
  color: var(--color-success);
  background: var(--color-success-soft);
  border-color: var(--color-success-border);
}

.badge-warning {
  color: var(--color-warning);
  background: var(--color-warning-soft);
  border-color: var(--color-warning-border);
}

.badge-danger {
  color: var(--color-danger);
  background: var(--color-danger-soft);
  border-color: var(--color-danger-border);
}

.badge-excellent {
  color: var(--grade-excellent);
  background: var(--grade-excellent-soft);
  border-color: var(--grade-excellent-border);
}

.badge-pass {
  color: var(--grade-pass);
  background: var(--grade-pass-soft);
  border-color: var(--grade-pass-border);
}

.badge-improve {
  color: var(--grade-improve);
  background: var(--grade-improve-soft);
  border-color: var(--grade-improve-border);
}
</style>
