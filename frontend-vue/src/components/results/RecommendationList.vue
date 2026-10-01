<template>
  <div class="recommendation-result">
    <section v-if="analysis" class="analysis" aria-label="分析总结">
      <h3 class="block-title">
        <AppIcon name="lightbulb" :size="16" />
        分析总结
      </h3>
      <p class="analysis-text plain-text">{{ analysis }}</p>
    </section>

    <section v-if="recommendations.length > 0" aria-label="推荐题目标签">
      <h3 class="block-title">
        <AppIcon name="target" :size="16" />
        推荐练习方向
        <span class="count">{{ recommendations.length }}</span>
      </h3>
      <ol class="rec-list">
        <li v-for="(rec, index) in recommendations" :key="index" class="rec-item">
          <div class="rec-header">
            <span class="rec-index" aria-hidden="true">{{ index + 1 }}</span>
            <span class="rec-tag">{{ rec.tag }}</span>
            <span :class="['rec-relevance', relevanceTone(rec.relevance)]">
              相关度 {{ formatRelevance(rec.relevance) }}
            </span>
          </div>
          <div
            class="meter"
            role="meter"
            aria-label="相关度"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-valuenow="percent(rec.relevance)"
          >
            <span :class="['meter-fill', relevanceTone(rec.relevance)]" :style="{ width: `${percent(rec.relevance)}%` }"></span>
          </div>
          <p v-if="rec.reason" class="rec-reason plain-text">{{ rec.reason }}</p>
        </li>
      </ol>
    </section>
  </div>
</template>

<script setup>
import AppIcon from '../ui/AppIcon.vue'

/** 题目推荐结果：分析总结 + 推荐标签列表（含相关度） */
defineProps({
  analysis: { type: String, default: '' },
  recommendations: { type: Array, default: () => [] }
})

const percent = (relevance) => {
  const n = Number(relevance)
  if (!Number.isFinite(n)) return 0
  return Math.max(0, Math.min(100, Math.round(n * 100)))
}

const formatRelevance = (relevance) => `${(relevance * 100).toFixed(0)}%`

const relevanceTone = (relevance) => {
  if (relevance >= 0.8) return 'relevance-high'
  if (relevance >= 0.5) return 'relevance-medium'
  return 'relevance-low'
}
</script>

<style scoped>
.recommendation-result {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.block-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
}

.count {
  padding: 0 7px;
  font-size: var(--fs-xs);
  line-height: 18px;
  color: var(--color-text-muted);
  background: var(--color-surface-3);
  border-radius: var(--radius-full);
}

.analysis {
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.analysis .block-title {
  margin-bottom: var(--space-2);
}

.analysis-text {
  font-size: var(--fs-md);
  line-height: 1.75;
}

.rec-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.rec-item {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.rec-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.rec-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border-radius: var(--radius-full);
}

.rec-tag {
  flex: 1;
  min-width: 0;
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  word-break: break-word;
}

.rec-relevance {
  flex-shrink: 0;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  font-variant-numeric: tabular-nums;
}

.rec-relevance.relevance-high {
  color: var(--color-success);
}

.rec-relevance.relevance-medium {
  color: var(--color-warning);
}

.rec-relevance.relevance-low {
  color: var(--color-text-muted);
}

.meter {
  height: 4px;
  margin: var(--space-3) 0 0 36px;
  overflow: hidden;
  background: var(--color-surface-3);
  border-radius: var(--radius-full);
}

.meter-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--color-text-muted);
}

.meter-fill.relevance-high {
  background: var(--color-success);
}

.meter-fill.relevance-medium {
  background: var(--color-warning);
}

.rec-reason {
  margin: var(--space-3) 0 0 36px;
  font-size: var(--fs-base);
  line-height: 1.7;
  color: var(--color-text-secondary);
}

@media (max-width: 640px) {
  .meter,
  .rec-reason {
    margin-left: 0;
  }

  .analysis {
    padding: var(--space-4);
  }
}
</style>
