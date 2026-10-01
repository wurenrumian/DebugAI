<template>
  <li class="record-item">
    <span class="record-icon" aria-hidden="true">
      <AppIcon :name="icon" :size="18" />
    </span>
    <div class="record-main">
      <h3 class="record-title" :title="fullTitle || undefined">{{ title }}</h3>
      <div class="record-meta">
        <time v-if="time" class="record-time" :datetime="isoTime">{{ time }}</time>
        <span v-for="(stat, i) in stats" :key="i" class="record-stat">{{ stat }}</span>
      </div>
    </div>
    <AppButton size="sm" icon-right="chevron-right" class="record-action" :aria-label="`查看详情：${title}`" @click="$emit('view')">
      查看详情
    </AppButton>
  </li>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import AppButton from '../ui/AppButton.vue'

/** 历史记录列表中的一行（调试 / 评价 / 推荐通用） */
const props = defineProps({
  icon: { type: String, default: 'file' },
  title: { type: String, required: true },
  fullTitle: { type: String, default: '' },
  time: { type: String, default: '' },
  timestamp: { type: [Number, String], default: null },
  stats: { type: Array, default: () => [] }
})

defineEmits(['view'])

const isoTime = computed(() => {
  if (props.timestamp === null || props.timestamp === undefined) return undefined
  const d = new Date(props.timestamp)
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
})
</script>

<style scoped>
.record-item {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface);
  transition: background-color var(--duration-fast) var(--ease);
}

.record-item + .record-item {
  border-top: 1px solid var(--color-border);
}

.record-item:hover {
  background: var(--color-surface-hover);
}

.record-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  border-radius: var(--radius-md);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
}

.record-main {
  flex: 1;
  min-width: 0;
}

.record-title {
  overflow: hidden;
  font-size: var(--fs-md);
  font-weight: var(--fw-medium);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.record-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 2px var(--space-3);
  margin-top: 2px;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.record-stat::before {
  content: '·';
  margin-right: var(--space-3);
  color: var(--color-border-strong);
}

.record-time + .record-stat::before,
.record-stat + .record-stat::before {
  display: inline;
}

.record-meta > .record-stat:first-child::before {
  display: none;
}

@media (max-width: 640px) {
  .record-item {
    flex-wrap: wrap;
    gap: var(--space-3);
    padding: var(--space-4);
  }

  .record-main {
    flex-basis: calc(100% - 54px);
  }

  .record-action {
    width: 100%;
  }
}
</style>
