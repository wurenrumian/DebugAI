<template>
  <component :is="as" :class="['card', `card-pad-${padding}`]" :aria-labelledby="title ? headingId : undefined">
    <header v-if="title || $slots.header || $slots.actions" class="card-header">
      <slot name="header">
        <div class="card-heading">
          <component :is="headingTag" :id="headingId" class="card-title">
            <AppIcon v-if="icon" :name="icon" :size="18" class="card-title-icon" />
            {{ title }}
          </component>
          <p v-if="description" class="card-description">{{ description }}</p>
        </div>
      </slot>
      <div v-if="$slots.actions" class="card-actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="card-body">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="card-footer">
      <slot name="footer" />
    </footer>
  </component>
</template>

<script setup>
import { computed } from 'vue'
import { uid } from '../../utils/uid'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  title: { type: String, default: '' },
  description: { type: String, default: '' },
  icon: { type: String, default: '' },
  // 标题层级，默认 h2（页面 h1 由 PageHeader 提供）
  level: { type: Number, default: 2 },
  as: { type: String, default: 'section' },
  // none | sm | md | lg
  padding: { type: String, default: 'md' }
})

const headingId = uid('card-title')
const headingTag = computed(() => `h${Math.min(Math.max(props.level, 2), 6)}`)
</script>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xs);
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
  padding: var(--space-4) var(--space-5) 0;
}

.card-heading {
  min-width: 0;
  flex: 1 1 auto;
}

.card-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--color-text);
}

.card-title-icon {
  color: var(--color-primary-text);
}

.card-description {
  margin-top: 2px;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.card-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.card-body {
  flex: 1 1 auto;
  min-width: 0;
  padding: var(--space-5);
}

.card-header + .card-body {
  padding-top: var(--space-4);
}

.card-pad-none > .card-body {
  padding: 0;
}

.card-pad-sm > .card-body {
  padding: var(--space-4);
}

.card-pad-lg > .card-body {
  padding: var(--space-6);
}

.card-pad-lg > .card-header {
  padding: var(--space-6) var(--space-6) 0;
}

.card-footer {
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--color-border);
}

@media (max-width: 640px) {
  .card-header {
    padding: var(--space-4) var(--space-4) 0;
  }

  .card-body,
  .card-pad-lg > .card-body {
    padding: var(--space-4);
  }

  .card-pad-lg > .card-header {
    padding: var(--space-4) var(--space-4) 0;
  }

  .card-footer {
    padding: var(--space-3) var(--space-4);
  }
}
</style>
