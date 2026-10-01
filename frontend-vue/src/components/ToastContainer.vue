<template>
  <div class="toast-container" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div
        v-for="item in toastState.items"
        :key="item.id"
        :class="['toast-item', `toast-${item.type}`]"
        :role="item.type === 'error' ? 'alert' : 'status'"
      >
        <AppIcon :name="iconFor(item.type)" :size="18" class="toast-icon" />
        <span class="toast-message">{{ item.message }}</span>
        <button
          type="button"
          class="toast-close"
          aria-label="关闭提示"
          @click="dismissToast(item.id)"
        >
          <AppIcon name="close" :size="15" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { toastState, dismissToast } from '../utils/toast'
import AppIcon from './ui/AppIcon.vue'

const iconFor = (type) => ({
  success: 'check-circle',
  error: 'alert',
  warning: 'alert',
  info: 'info'
}[type] || 'info')
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: calc(var(--nav-height) + 12px);
  left: 50%;
  transform: translateX(-50%);
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  pointer-events: none;
  width: max-content;
  max-width: calc(100vw - 32px);
}

.toast-item {
  pointer-events: auto;
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  min-width: 240px;
  max-width: 520px;
  padding: 12px 12px 12px 14px;
  font-size: var(--fs-base);
  line-height: 1.5;
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-left-width: 4px;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  word-break: break-word;
}

.toast-icon {
  margin-top: 1px;
}

.toast-message {
  flex: 1;
  min-width: 0;
}

.toast-success {
  border-left-color: var(--color-success);
}

.toast-success .toast-icon {
  color: var(--color-success);
}

.toast-error {
  border-left-color: var(--color-danger);
}

.toast-error .toast-icon {
  color: var(--color-danger);
}

.toast-warning {
  border-left-color: var(--color-warning);
}

.toast-warning .toast-icon {
  color: var(--color-warning);
}

.toast-info {
  border-left-color: var(--color-info);
}

.toast-info .toast-icon {
  color: var(--color-info);
}

.toast-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
}

.toast-close:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.25s, transform 0.25s;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

@media (max-width: 640px) {
  .toast-item {
    min-width: 0;
    width: calc(100vw - 32px);
  }
}
</style>
