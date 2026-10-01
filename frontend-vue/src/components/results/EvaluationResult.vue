<template>
  <div class="evaluation-result">
    <section v-if="result.overall_evaluation" class="overall" aria-label="整体评价">
      <h3 class="overall-title">
        <AppIcon name="file" :size="16" />
        整体评价
      </h3>
      <p class="overall-text plain-text">{{ result.overall_evaluation }}</p>
    </section>

    <div class="dimension-grid">
      <article
        v-for="dim in visibleDimensions"
        :key="dim.key"
        :class="['dimension', `tone-${gradeTone(result[dim.key]?.grade)}`]"
      >
        <header class="dimension-header">
          <h3 class="dimension-title">{{ dim.label }}</h3>
          <GradeBadge :grade="result[dim.key]?.grade" size="md" />
        </header>
        <p v-if="result[dim.key]?.analysis" class="dimension-analysis plain-text">{{ result[dim.key].analysis }}</p>
        <p v-else class="dimension-analysis is-empty">暂无分析</p>
      </article>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import GradeBadge from '../ui/GradeBadge.vue'
import { gradeMeta } from '../ui/grades'

/** 代码评价结果：整体评价 + 四个维度（功能正确 / 逻辑严谨 / 算法效率 / 结构规范） */
const props = defineProps({
  result: { type: Object, required: true },
  // 为 true 时只显示结果中存在的维度（历史详情）
  onlyPresent: { type: Boolean, default: false }
})

const DIMENSIONS = [
  { key: 'functional_correctness', label: '功能正确' },
  { key: 'logical_rigor', label: '逻辑严谨' },
  { key: 'algorithm_quality', label: '算法效率' },
  { key: 'structural_normativity', label: '结构规范' }
]

const visibleDimensions = computed(() =>
  props.onlyPresent ? DIMENSIONS.filter(d => props.result[d.key]) : DIMENSIONS
)

const gradeTone = (grade) => gradeMeta(grade).tone
</script>

<style scoped>
.evaluation-result {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.overall {
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.overall-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
  font-size: var(--fs-base);
  color: var(--color-text-secondary);
}

.overall-text {
  font-size: var(--fs-md);
  line-height: 1.75;
  color: var(--color-text);
}

.dimension-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
}

.dimension {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  overflow: hidden;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.dimension::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--color-border-strong);
}

.dimension.tone-excellent::before {
  background: var(--grade-excellent);
}

.dimension.tone-pass::before {
  background: var(--grade-pass);
}

.dimension.tone-improve::before {
  background: var(--grade-improve);
}

.dimension-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.dimension-title {
  font-size: var(--fs-md);
}

.dimension-analysis {
  font-size: var(--fs-base);
  line-height: 1.7;
  color: var(--color-text-secondary);
}

.dimension-analysis.is-empty {
  color: var(--color-text-muted);
}

@media (max-width: 640px) {
  .dimension-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .overall {
    padding: var(--space-4);
  }
}
</style>
