<template>
  <div :class="['alert', `alert-${tone}`]" :role="tone === 'danger' ? 'alert' : 'status'">
    <AppIcon :name="iconName" :size="18" class="alert-icon" />
    <div class="alert-body">
      <p v-if="title" class="alert-title">{{ title }}</p>
      <div class="alert-content"><slot /></div>
    </div>
    <button
      v-if="dismissible"
      type="button"
      class="alert-close"
      aria-label="关闭提示"
      @click="$emit('dismiss')"
    >
      <AppIcon name="close" :size="16" />
    </button>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  // info | success | warning | danger
  tone: { type: String, default: 'info' },
  title: { type: String, default: '' },
  dismissible: { type: Boolean, default: false }
})

defineEmits(['dismiss'])

const iconName = computed(() => ({
  success: 'check-circle',
  warning: 'alert',
  danger: 'alert',
  info: 'info'
}[props.tone] || 'info'))
</script>

<style scoped>
.alert {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  font-size: var(--fs-base);
  line-height: 1.6;
  border: 1px solid;
  border-radius: var(--radius-md);
}

.alert-icon {
  margin-top: 2px;
}

.alert-body {
  flex: 1;
  min-width: 0;
  color: var(--color-text);
  word-break: break-word;
}

.alert-title {
  font-weight: var(--fw-semibold);
  margin-bottom: 2px;
}

.alert-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  margin: -2px -6px -2px 0;
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
}

.alert-close:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.alert-info {
  background: var(--color-info-soft);
  border-color: var(--color-info-border);
}

.alert-info .alert-icon {
  color: var(--color-info);
}

.alert-success {
  background: var(--color-success-soft);
  border-color: var(--color-success-border);
}

.alert-success .alert-icon {
  color: var(--color-success);
}

.alert-warning {
  background: var(--color-warning-soft);
  border-color: var(--color-warning-border);
}

.alert-warning .alert-icon {
  color: var(--color-warning);
}

.alert-danger {
  background: var(--color-danger-soft);
  border-color: var(--color-danger-border);
}

.alert-danger .alert-icon {
  color: var(--color-danger);
}
</style>
