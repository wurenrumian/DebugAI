<template>
  <component
    :is="tag"
    :class="classes"
    v-bind="linkOrButtonAttrs"
  >
    <AppSpinner v-if="loading" size="sm" />
    <AppIcon v-else-if="icon" :name="icon" :size="iconSize" />
    <span v-if="$slots.default" class="btn-label"><slot /></span>
    <AppIcon v-if="iconRight && !loading" :name="iconRight" :size="iconSize" />
  </component>
</template>

<script setup>
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import AppIcon from './AppIcon.vue'
import AppSpinner from './AppSpinner.vue'

const props = defineProps({
  // primary | secondary | ghost | danger | danger-ghost | link
  variant: { type: String, default: 'secondary' },
  // sm | md | lg
  size: { type: String, default: 'md' },
  type: { type: String, default: 'button' },
  to: { type: [String, Object], default: null },
  disabled: { type: Boolean, default: false },
  loading: { type: Boolean, default: false },
  block: { type: Boolean, default: false },
  icon: { type: String, default: '' },
  iconRight: { type: String, default: '' },
  // 仅图标按钮（必须同时提供 aria-label）
  iconOnly: { type: Boolean, default: false }
})

const tag = computed(() => (props.to ? RouterLink : 'button'))

const iconSize = computed(() => (props.size === 'sm' ? 15 : props.size === 'lg' ? 18 : 16))

const classes = computed(() => [
  'btn',
  `btn-${props.variant}`,
  `btn-${props.size}`,
  {
    'btn-block': props.block,
    'btn-icon-only': props.iconOnly,
    'is-loading': props.loading
  }
])

const linkOrButtonAttrs = computed(() => {
  if (props.to) {
    return { to: props.to }
  }
  return {
    type: props.type,
    disabled: props.disabled || props.loading,
    'aria-busy': props.loading ? 'true' : undefined
  }
})
</script>

<style scoped>
.btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: var(--control-height);
  padding: 0 16px;
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  line-height: 1.2;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  white-space: nowrap;
  text-decoration: none;
  user-select: none;
  transition:
    background-color var(--duration-fast) var(--ease),
    border-color var(--duration-fast) var(--ease),
    color var(--duration-fast) var(--ease),
    box-shadow var(--duration-fast) var(--ease);
}

.btn:hover {
  text-decoration: none;
}

.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.btn-sm {
  min-height: var(--control-height-sm);
  padding: 0 12px;
  font-size: var(--fs-sm);
  border-radius: var(--radius-sm);
}

.btn-lg {
  min-height: var(--control-height-lg);
  padding: 0 22px;
  font-size: var(--fs-md);
}

.btn-block {
  display: flex;
  width: 100%;
}

.btn-icon-only {
  padding: 0;
  width: var(--control-height);
}

.btn-icon-only.btn-sm {
  width: var(--control-height-sm);
}

.btn-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* primary */
.btn-primary {
  color: var(--color-on-primary);
  background: var(--color-primary);
  box-shadow: var(--shadow-xs);
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.btn-primary:active:not(:disabled) {
  background: var(--color-primary-active);
}

/* secondary */
.btn-secondary {
  color: var(--color-text);
  background: var(--color-surface);
  border-color: var(--color-border-strong);
  box-shadow: var(--shadow-xs);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-surface-hover);
  border-color: var(--color-text-muted);
}

/* ghost */
.btn-ghost {
  color: var(--color-text-secondary);
  background: transparent;
}

.btn-ghost:hover:not(:disabled) {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

/* danger */
.btn-danger {
  color: var(--color-on-solid);
  background: var(--color-danger);
}

.btn-danger:hover:not(:disabled) {
  background: var(--color-danger-hover);
}

.btn-danger-ghost {
  color: var(--color-danger);
  background: transparent;
  border-color: var(--color-danger-border);
}

.btn-danger-ghost:hover:not(:disabled) {
  background: var(--color-danger-soft);
}

/* link */
.btn-link {
  min-height: auto;
  padding: 0;
  color: var(--color-primary-text);
  background: transparent;
  border: none;
}

.btn-link:hover:not(:disabled) {
  text-decoration: underline;
  text-underline-offset: 2px;
}
</style>
