<template>
  <div class="toast-container" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <div
        v-for="item in toastState.items"
        :key="item.id"
        :class="['toast-item', `toast-${item.type}`]"
        :role="item.type === 'error' ? 'alert' : 'status'"
      >
        <span class="toast-message">{{ item.message }}</span>
        <button
          type="button"
          class="toast-close"
          aria-label="关闭提示"
          @click="dismissToast(item.id)"
        >×</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { toastState, dismissToast } from '../utils/toast'
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 3000;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  pointer-events: none;
  width: max-content;
  max-width: calc(100vw - 32px);
}

.toast-item {
  pointer-events: auto;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px 12px 20px;
  border-radius: 6px;
  color: white;
  font-size: 14px;
  line-height: 1.5;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  word-break: break-word;
}

.toast-success {
  background: #67c23a;
}

.toast-error {
  background: #f56c6c;
}

.toast-warning {
  background: #e6a23c;
}

.toast-info {
  background: #409eff;
}

.toast-close {
  background: transparent;
  color: inherit;
  font-size: 18px;
  line-height: 1;
  padding: 0 4px;
  opacity: 0.8;
}

.toast-close:hover {
  opacity: 1;
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
</style>
