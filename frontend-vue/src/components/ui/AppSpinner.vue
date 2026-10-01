<template>
  <span
    :class="['spinner-wrap', { 'spinner-block': label && block }]"
    :role="label ? 'status' : undefined"
  >
    <span :class="['spinner', `spinner-${size}`]" aria-hidden="true"></span>
    <span v-if="label" :class="block ? 'spinner-label' : 'sr-only'">{{ label }}</span>
  </span>
</template>

<script setup>
defineProps({
  size: { type: String, default: 'md' }, // sm | md | lg
  // 有 label 时作为 role="status" 播报；block 时文字可见并居中显示
  label: { type: String, default: '' },
  block: { type: Boolean, default: false }
})
</script>

<style scoped>
.spinner-wrap {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.spinner-block {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: var(--space-3);
  padding: var(--space-12) var(--space-4);
  color: var(--color-text-muted);
  font-size: var(--fs-sm);
}

.spinner {
  display: inline-block;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-right-color: transparent;
  animation: spin 0.75s linear infinite;
  opacity: 0.85;
}

.spinner-block .spinner {
  color: var(--color-primary-text);
}

.spinner-sm {
  width: 14px;
  height: 14px;
}

.spinner-md {
  width: 20px;
  height: 20px;
}

.spinner-lg {
  width: 28px;
  height: 28px;
  border-width: 3px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
