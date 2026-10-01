<template>
  <div class="skeleton" aria-hidden="true">
    <span
      v-for="n in lines"
      :key="n"
      class="skeleton-line"
      :style="{ width: n === lines && lines > 1 ? '60%' : '100%', height: lineHeight }"
    ></span>
  </div>
</template>

<script setup>
defineProps({
  lines: { type: Number, default: 3 },
  lineHeight: { type: String, default: '12px' }
})
</script>

<style scoped>
.skeleton {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.skeleton-line {
  display: block;
  border-radius: var(--radius-sm);
  background: linear-gradient(
    90deg,
    var(--color-surface-3) 25%,
    var(--color-surface-hover) 50%,
    var(--color-surface-3) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.4s ease-in-out infinite;
}

@keyframes shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
