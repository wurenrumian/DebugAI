<template>
  <div class="filter-panel">
    <div class="filter-grid">
      <div class="field field-students">
        <label class="field-label" :for="`${idPrefix}-students`">学生筛选</label>
        <select
          :id="`${idPrefix}-students`"
          v-model="studentsModel"
          multiple
          class="multi-select"
          :aria-describedby="`${idPrefix}-students-hint`"
        >
          <option v-for="student in students" :key="student.student_id" :value="student.student_id">
            {{ student.username }} ({{ student.student_id }})
          </option>
        </select>
        <p :id="`${idPrefix}-students-hint`" class="field-hint">
          按住 Ctrl/Cmd 可多选，不选默认为全班
          <template v-if="selectedStudents.length > 0">
            · 已选 {{ selectedStudents.length }} 人
            <button type="button" class="clear-btn" @click="$emit('update:selectedStudents', [])">清除</button>
          </template>
        </p>
      </div>

      <fieldset class="field field-dates">
        <legend class="field-label">时间范围</legend>
        <div class="date-range">
          <label class="sr-only" :for="`${idPrefix}-start`">开始日期</label>
          <input
            :id="`${idPrefix}-start`"
            v-model="startModel"
            type="date"
          />
          <span class="date-separator" aria-hidden="true">至</span>
          <label class="sr-only" :for="`${idPrefix}-end`">结束日期</label>
          <input
            :id="`${idPrefix}-end`"
            v-model="endModel"
            type="date"
          />
        </div>
      </fieldset>
    </div>

    <div class="filter-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { uid } from '../../utils/uid'

/** 班级历史 / 班级薄弱点共用的筛选条件（学生多选 + 日期范围） */
const props = defineProps({
  students: { type: Array, default: () => [] },
  selectedStudents: { type: Array, default: () => [] },
  startDate: { type: String, default: '' },
  endDate: { type: String, default: '' }
})

const emit = defineEmits(['update:selectedStudents', 'update:startDate', 'update:endDate'])

const idPrefix = uid('class-filter')

const studentsModel = computed({
  get: () => props.selectedStudents,
  set: (value) => emit('update:selectedStudents', value)
})
const startModel = computed({
  get: () => props.startDate,
  set: (value) => emit('update:startDate', value)
})
const endModel = computed({
  get: () => props.endDate,
  set: (value) => emit('update:endDate', value)
})
</script>

<style scoped>
.filter-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-5);
  margin-bottom: var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.filter-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: var(--space-4);
}

.filter-grid .field + .field {
  margin-top: 0;
}

fieldset {
  border: none;
  min-width: 0;
}

legend.field-label {
  margin-bottom: 6px;
  padding: 0;
}

.multi-select {
  min-height: 108px;
}

.date-range {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.date-range input {
  flex: 1;
  min-width: 0;
}

.date-separator {
  flex-shrink: 0;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.clear-btn {
  margin-left: 4px;
  font-size: var(--fs-xs);
  color: var(--color-primary-text);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.filter-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

@media (max-width: 760px) {
  .filter-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .filter-panel {
    padding: var(--space-4);
  }

  .filter-actions > * {
    flex: 1 1 auto;
  }
}
</style>
