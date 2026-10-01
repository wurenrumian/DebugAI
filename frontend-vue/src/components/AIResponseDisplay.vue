<template>
  <div class="ai-response-display">
    <!-- 学生回复（若非空） -->
    <div v-if="studentResponse" class="section student-response-section">
      <h4 class="section-title"><AppIcon name="user" :size="14" />你的回复</h4>
      <div class="section-content plain-text">{{ studentResponse }}</div>
    </div>

    <!-- AI 回复 -->
    <div v-if="aiResponse" class="ai-response-content">
      <!-- 纯文本显示（当无法解析为结构化 JSON 时） -->
      <div v-if="aiResponse.content && !isStructured" class="section">
        <div class="section-content markdown-body" v-html="renderMarkdown(aiResponse.content)"></div>
      </div>

      <!-- 学生思路 -->
      <div v-if="aiResponse.student_thought" class="section">
        <h4 class="section-title"><AppIcon name="chat" :size="14" />你的思路</h4>
        <div class="section-content plain-text">{{ aiResponse.student_thought }}</div>
      </div>

      <!-- 建议修正 -->
      <div v-if="aiResponse.suggested_correction" class="section">
        <h4 class="section-title"><AppIcon name="check" :size="14" />建议修正</h4>
        <div class="section-content plain-text">{{ aiResponse.suggested_correction }}</div>
      </div>

      <!-- 问题总结 -->
      <div v-if="aiResponse.problem_summary" class="section">
        <h4 class="section-title"><AppIcon name="file" :size="14" />问题总结</h4>
        <div class="section-content plain-text">{{ aiResponse.problem_summary }}</div>
      </div>

      <!-- 关键问题列表 -->
      <div v-if="normalizedIssues && normalizedIssues.length > 0" class="section">
        <h4 class="section-title section-title-warning"><AppIcon name="alert" :size="14" />关键问题</h4>
        <ul class="section-content issue-list">
          <li
            v-for="(issue, index) in normalizedIssues"
            :key="index"
            class="issue-item"
          >
            <div class="issue-description">{{ issue.description }}</div>
            <div v-if="issue.location" class="issue-location">
              <span class="location-label">位置</span>
              <code>{{ issue.location }}</code>
            </div>
          </li>
        </ul>
      </div>

      <!-- 薄弱点 -->
      <div v-if="normalizedWeakpoints && normalizedWeakpoints.length > 0" class="section">
        <h4 class="section-title"><AppIcon name="target" :size="14" />薄弱点</h4>
        <div class="section-content weakpoints-list">
          <span
            v-for="(wp, index) in normalizedWeakpoints"
            :key="index"
            class="weakpoint-tag"
          >
            {{ wp }}
          </span>
        </div>
      </div>

      <!-- 调试指导 -->
      <div v-if="aiResponse.debug_guidance" class="section">
        <h4 class="section-title"><AppIcon name="lightbulb" :size="14" />调试指导</h4>
        <div class="section-content guidance-content markdown-body" v-html="renderMarkdown(aiResponse.debug_guidance)"></div>
      </div>

      <!-- 详细请求 -->
      <div v-if="aiResponse.ask_for_detail" class="section section-callout">
        <h4 class="section-title"><AppIcon name="help" :size="14" />请补充</h4>
        <div class="section-content plain-text">{{ aiResponse.ask_for_detail }}</div>
      </div>

      <!-- 求助内容 -->
      <div v-if="aiResponse.ask_for_help" class="section section-callout">
        <h4 class="section-title"><AppIcon name="help" :size="14" />需要帮助</h4>
        <div class="section-content plain-text">{{ aiResponse.ask_for_help }}</div>
      </div>

      <!-- 建议列表 -->
      <div v-if="aiResponse.suggestions && aiResponse.suggestions.length > 0" class="section">
        <h4 class="section-title"><AppIcon name="sparkles" :size="14" />建议</h4>
        <ul class="section-content suggestions-list">
          <li v-for="(suggestion, index) in aiResponse.suggestions" :key="index">
            <div class="markdown-body" v-html="renderMarkdown(suggestion)"></div>
          </li>
        </ul>
      </div>
    </div>

    <!-- 空状态 -->
    <p v-else class="empty-state">暂无 AI 回复</p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '../utils/markdown'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({
  aiResponse: {
    type: Object,
    default: null
  },
  studentResponse: {
    type: String,
    default: ''
  }
})

// 判断是否为结构化响应
const isStructured = computed(() => {
  if (!props.aiResponse) return false
  const keys = [
    'student_thought', 'suggested_correction', 'problem_summary',
    'key_issues', 'keyIssues', 'weak_points', 'weakpoints',
    'debug_guidance', 'suggestions'
  ]
  return keys.some(key => props.aiResponse[key] !== undefined)
})

// 处理 key_issues（兼容不同命名）
const normalizedIssues = computed(() => {
  if (!props.aiResponse) return []
  return props.aiResponse.key_issues || props.aiResponse.keyIssues || []
})

// 处理 weakpoints（支持 weakpoints 或 weak_points）
const normalizedWeakpoints = computed(() => {
  if (!props.aiResponse) return []
  return props.aiResponse.weakpoints || props.aiResponse.weak_points || []
})
</script>

<style scoped>
.ai-response-display {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  font-size: var(--fs-base);
  line-height: 1.7;
}

.ai-response-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
}

.section-title .app-icon {
  color: var(--color-primary-text);
}

.section-title-warning .app-icon {
  color: var(--color-warning);
}

.section-content {
  color: var(--color-text);
  word-break: break-word;
}

.student-response-section {
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px dashed var(--color-border-strong);
  border-radius: var(--radius-md);
}

.section-callout {
  padding: var(--space-3);
  background: var(--color-warning-soft);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-md);
}

.section-callout .section-title .app-icon {
  color: var(--color-warning);
}

.issue-list {
  display: flex;
  flex-direction: column;
}

.issue-item {
  padding: 6px 0;
}

.issue-item:not(:last-child) {
  border-bottom: 1px dashed var(--color-border);
}

.issue-item:first-child {
  padding-top: 0;
}

.issue-location {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 2px;
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

.issue-location code {
  padding: 1px 6px;
  font-size: var(--fs-xs);
  color: var(--code-inline-text);
  background: var(--code-inline-bg);
  border-radius: 4px;
  word-break: break-all;
}

.weakpoints-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.weakpoint-tag {
  display: inline-block;
  padding: 1px 10px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--color-warning);
  background: var(--color-warning-soft);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-full);
}

.suggestions-list {
  padding-left: 20px;
  list-style: decimal;
}

.suggestions-list li {
  margin: 4px 0;
}

.suggestions-list li::marker {
  color: var(--color-text-muted);
}

.empty-state {
  padding: var(--space-4);
  text-align: center;
  color: var(--color-text-muted);
}
</style>
