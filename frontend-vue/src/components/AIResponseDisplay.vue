<template>
  <div class="ai-response-display">
    <!-- 学生回复（若非空） -->
    <div v-if="studentResponse" class="section student-response-section">
      <div class="section-header">
        <span class="section-icon">👤</span>
        <span class="section-title">你的回复</span>
      </div>
      <div class="section-content">
        {{ studentResponse }}
      </div>
    </div>

    <!-- AI 回复 -->
    <div v-if="aiResponse" class="ai-response-content">
      <!-- 纯文本显示（当无法解析为结构化 JSON 时） -->
      <div v-if="aiResponse.content && !isStructured" class="section">
        <div class="section-header">
          <span class="section-icon">🤖</span>
          <span class="section-title">AI 助手</span>
        </div>
        <div class="section-content" v-html="formatContent(aiResponse.content)"></div>
      </div>

      <!-- 学生思路 -->
      <div v-if="aiResponse.student_thought" class="section">
        <div class="section-header">
          <span class="section-icon">💭</span>
          <span class="section-title">你的思路</span>
        </div>
        <div class="section-content">
          {{ aiResponse.student_thought }}
        </div>
      </div>

      <!-- 建议修正 -->
      <div v-if="aiResponse.suggested_correction" class="section">
        <div class="section-header">
          <span class="section-icon">✏️</span>
          <span class="section-title">建议修正</span>
        </div>
        <div class="section-content">
          {{ aiResponse.suggested_correction }}
        </div>
      </div>

      <!-- 问题总结 -->
      <div v-if="aiResponse.problem_summary" class="section">
        <div class="section-header">
          <span class="section-icon">📋</span>
          <span class="section-title">问题总结</span>
        </div>
        <div class="section-content">
          {{ aiResponse.problem_summary }}
        </div>
      </div>

      <!-- 关键问题列表 -->
      <div v-if="normalizedIssues && normalizedIssues.length > 0" class="section">
        <div class="section-header">
          <span class="section-icon">⚠️</span>
          <span class="section-title">关键问题</span>
        </div>
        <div class="section-content">
          <div
            v-for="(issue, index) in normalizedIssues"
            :key="index"
            class="issue-item"
          >
            <div class="issue-description">{{ issue.description }}</div>
            <div v-if="issue.location" class="issue-location">
              <span class="location-label">位置：</span>
              <code>{{ issue.location }}</code>
            </div>
          </div>
        </div>
      </div>

      <!-- 薄弱点 -->
      <div v-if="normalizedWeakpoints && normalizedWeakpoints.length > 0" class="section">
        <div class="section-header">
          <span class="section-icon">🎯</span>
          <span class="section-title">薄弱点</span>
        </div>
        <div class="section-content">
          <div class="weakpoints-list">
            <span
              v-for="(wp, index) in normalizedWeakpoints"
              :key="index"
              class="weakpoint-tag"
            >
              {{ wp }}
            </span>
          </div>
        </div>
      </div>

      <!-- 调试指导 -->
      <div v-if="aiResponse.debug_guidance" class="section">
        <div class="section-header">
          <span class="section-icon">🔧</span>
          <span class="section-title">调试指导</span>
        </div>
        <div class="section-content guidance-content" v-html="formatContent(aiResponse.debug_guidance)"></div>
      </div>

      <!-- 详细请求 -->
      <div v-if="aiResponse.ask_for_detail" class="section">
        <div class="section-header">
          <span class="section-icon">❓</span>
          <span class="section-title">请补充</span>
        </div>
        <div class="section-content">
          {{ aiResponse.ask_for_detail }}
        </div>
      </div>

      <!-- 求助内容 -->
      <div v-if="aiResponse.ask_for_help" class="section">
        <div class="section-header">
          <span class="section-icon">🆘</span>
          <span class="section-title">需要帮助</span>
        </div>
        <div class="section-content">
          {{ aiResponse.ask_for_help }}
        </div>
      </div>

      <!-- 建议列表 -->
      <div v-if="aiResponse.suggestions && aiResponse.suggestions.length > 0" class="section">
        <div class="section-header">
          <span class="section-icon">💡</span>
          <span class="section-title">建议</span>
        </div>
        <div class="section-content">
          <ul class="suggestions-list">
            <li v-for="(suggestion, index) in aiResponse.suggestions" :key="index">
              <div v-html="formatContent(suggestion)"></div>
            </li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="empty-state">
      暂无 AI 回复
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

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

// 格式化内容（支持简单的 Markdown）
const formatContent = (content) => {
  if (!content) return ''
  
  let formatted = content
    .replace(/&/g, '&')
    .replace(/</g, '<')
    .replace(/>/g, '>')
  
  // 代码块
  formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
  formatted = formatted.replace(/`([^`]+)`/g, '<code>$1</code>')
  
  // 换行
  formatted = formatted.replace(/\n/g, '<br>')
  
  // 列表项标记
  formatted = formatted.replace(/^- (.+)$/gm, '<li>$1</li>')
  formatted = formatted.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
  formatted = formatted.replace(/<\/ul>\s*<ul>/g, '')
  
  return formatted
}
</script>

<style scoped>
.ai-response-display {
  font-size: 14px;
  line-height: 1.6;
}

.section {
  margin-bottom: 12px;
  padding: 8px 0;
}

.section:last-child {
  margin-bottom: 0;
}

.student-response-section {
  padding: 8px 12px;
  background: #f0f9eb;
  border-radius: 6px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.section-icon {
  font-size: 14px;
}

.section-title {
  font-weight: 600;
  color: #303133;
  font-size: 14px;
}

.section-content {
  color: #606266;
  word-break: break-word;
  padding-left: 22px;
}

.issue-item {
  padding: 6px 0;
}

.issue-item:not(:last-child) {
  border-bottom: 1px dashed #ebeef5;
}

.issue-description {
  color: #303133;
  margin-bottom: 2px;
}

.issue-location {
  font-size: 12px;
  color: #909399;
}

.issue-location code {
  background: #f5f7fa;
  padding: 1px 4px;
  border-radius: 2px;
  font-family: 'Consolas', monospace;
  color: #e6a23c;
}

.weakpoints-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.weakpoint-tag {
  display: inline-block;
  padding: 2px 8px;
  background: #f56c6c;
  color: white;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.guidance-content :deep(pre) {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}

.guidance-content :deep(code) {
  font-family: 'Consolas', 'Monaco', monospace;
}

.guidance-content :deep(ul) {
  margin: 8px 0;
  padding-left: 20px;
}

.guidance-content :deep(li) {
  margin: 4px 0;
}

.suggestions-list {
  margin: 0;
  padding-left: 20px;
}

.suggestions-list li {
  margin: 8px 0;
  color: #606266;
}

.suggestions-list li :deep(pre) {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}

.suggestions-list li :deep(code) {
  font-family: 'Consolas', 'Monaco', monospace;
}

.empty-state {
  text-align: center;
  color: #909399;
  padding: 20px;
}
</style>