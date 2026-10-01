<template>
  <header class="page-header">
    <div class="page-header-text">
      <h1 class="page-title">{{ title }}</h1>
      <p v-if="description" class="page-description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header-actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' }
})
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
  margin-bottom: var(--space-6);
}

.page-header-text {
  min-width: 0;
}

.page-title {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.01em;
}

.page-description {
  margin-top: var(--space-1);
  font-size: var(--fs-base);
  color: var(--color-text-muted);
}

.page-header-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

@media (max-width: 640px) {
  .page-header {
    margin-bottom: var(--space-4);
  }

  .page-title {
    font-size: var(--fs-xl);
  }

  .page-description {
    font-size: var(--fs-sm);
  }
}
</style>
