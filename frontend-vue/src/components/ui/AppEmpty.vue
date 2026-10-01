<template>
  <div :class="['empty', `empty-${tone}`, { 'empty-compact': compact }]">
    <div class="empty-icon" aria-hidden="true">
      <AppIcon :name="icon" :size="compact ? 22 : 28" :stroke-width="1.6" />
    </div>
    <component :is="titleTag" v-if="title" class="empty-title">{{ title }}</component>
    <p v-if="description" class="empty-description">{{ description }}</p>
    <slot />
    <div v-if="$slots.actions" class="empty-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup>
import AppIcon from './AppIcon.vue'

defineProps({
  icon: { type: String, default: 'inbox' },
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  // neutral | danger
  tone: { type: String, default: 'neutral' },
  compact: { type: Boolean, default: false },
  titleTag: { type: String, default: 'p' }
})
</script>

<style scoped>
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: var(--space-12) var(--space-6);
  color: var(--color-text-muted);
}

.empty-compact {
  padding: var(--space-6) var(--space-4);
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin-bottom: var(--space-4);
  border-radius: var(--radius-full);
  color: var(--color-text-muted);
  background: var(--color-surface-3);
}

.empty-compact .empty-icon {
  width: 44px;
  height: 44px;
  margin-bottom: var(--space-3);
}

.empty-danger .empty-icon {
  color: var(--color-danger);
  background: var(--color-danger-soft);
}

.empty-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--color-text);
}

.empty-description {
  max-width: 420px;
  margin-top: var(--space-1);
  font-size: var(--fs-sm);
  line-height: 1.6;
}

.empty-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--space-2);
  margin-top: var(--space-5);
}
</style>
