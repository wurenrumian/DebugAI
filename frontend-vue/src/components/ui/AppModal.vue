<template>
  <Teleport to="body">
    <div class="modal-root">
      <div class="modal-overlay" aria-hidden="true" @click="onOverlayClick"></div>
      <div class="modal-scroller" @mousedown.self="onOverlayClick">
        <div
          ref="dialogRef"
          :class="['modal', `modal-${size}`]"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          tabindex="-1"
        >
          <header class="modal-header">
            <h2 :id="titleId" class="modal-title">{{ title }}</h2>
            <button
              type="button"
              class="modal-close"
              aria-label="关闭对话框"
              @click="$emit('close')"
            >
              <AppIcon name="close" :size="18" />
            </button>
          </header>
          <div class="modal-body">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="modal-footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import AppIcon from './AppIcon.vue'
import { getFocusable, trapTabKey } from '../../utils/focus'
import { uid } from '../../utils/uid'

/**
 * 模态对话框：由父组件 v-if 控制显示，关闭时触发 close 事件
 *  - Esc 关闭、点击遮罩关闭（可关闭）
 *  - 打开时焦点移入并限制在对话框内，关闭后恢复到触发元素
 *  - 打开期间锁定页面滚动
 */
const props = defineProps({
  title: { type: String, required: true },
  // sm | md | lg
  size: { type: String, default: 'md' },
  closeOnOverlay: { type: Boolean, default: true },
  // 打开时优先聚焦的元素选择器（默认第一个可聚焦元素之后的输入控件 / 对话框本身）
  initialFocus: { type: String, default: '' }
})

const emit = defineEmits(['close'])

const dialogRef = ref(null)
const titleId = uid('modal-title')
let previouslyFocused = null
let previousOverflow = ''

const onOverlayClick = () => {
  if (props.closeOnOverlay) emit('close')
}

const onKeydown = (event) => {
  if (event.key === 'Escape') {
    event.stopPropagation()
    emit('close')
    return
  }
  trapTabKey(event, dialogRef.value)
}

onMounted(async () => {
  previouslyFocused = document.activeElement
  previousOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  document.addEventListener('keydown', onKeydown, true)
  await nextTick()
  const dialog = dialogRef.value
  if (!dialog) return
  let target = props.initialFocus ? dialog.querySelector(props.initialFocus) : null
  if (!target) {
    // 跳过右上角关闭按钮，优先聚焦内容中的第一个表单控件
    target = getFocusable(dialog).find(el => /^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName)) || dialog
  }
  target.focus()
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown, true)
  document.body.style.overflow = previousOverflow
  if (previouslyFocused && typeof previouslyFocused.focus === 'function') {
    previouslyFocused.focus()
  }
})
</script>

<style scoped>
.modal-root {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
}

.modal-overlay {
  position: absolute;
  inset: 0;
  background: var(--color-overlay);
  animation: fade-in var(--duration) var(--ease);
}

.modal-scroller {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 6vh var(--space-4) var(--space-6);
  overflow-y: auto;
}

.modal {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-height: 88vh;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  animation: pop-in var(--duration) var(--ease);
}

.modal:focus {
  outline: none;
}

.modal-sm {
  max-width: 420px;
}

.modal-md {
  max-width: 600px;
}

.modal-lg {
  max-width: 880px;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.modal-title {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
}

.modal-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  margin-right: -6px;
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
}

.modal-close:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.modal-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-5);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5);
  border-top: 1px solid var(--color-border);
}

@media (max-width: 640px) {
  .modal-scroller {
    padding: var(--space-3);
    align-items: flex-end;
  }

  .modal {
    max-height: calc(100vh - 24px);
    max-height: calc(100dvh - 24px);
  }

  .modal-body {
    padding: var(--space-4);
  }

  .modal-header,
  .modal-footer {
    padding-left: var(--space-4);
    padding-right: var(--space-4);
  }
}

@keyframes fade-in {
  from {
    opacity: 0;
  }
}

@keyframes pop-in {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.98);
  }
}
</style>
