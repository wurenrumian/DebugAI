<template>
  <AppModal :title="modalTitle" size="lg" @close="$emit('close')">
    <!-- 首次提交的题目描述和代码（debug/evaluate类型显示） -->
    <section v-if="initialSubmission && type !== 'recommend'" class="initial-submission">
      <button
        type="button"
        class="submission-toggle"
        :aria-expanded="showCodeModal ? 'true' : 'false'"
        aria-controls="submission-content"
        @click="showCodeModal = !showCodeModal"
      >
        <AppIcon name="file" :size="16" />
        <span class="submission-title">题目描述与提交的代码</span>
        <AppIcon name="chevron-down" :size="16" :class="['expand-icon', { open: showCodeModal }]" />
      </button>
      <div v-if="showCodeModal" id="submission-content" class="submission-content">
        <div class="submission-block">
          <h3 class="block-label">题目</h3>
          <!-- 学生提交的内容一律按纯文本显示（教师端也会打开此弹窗） -->
          <div class="problem-text plain-text">{{ initialSubmission.problem_description }}</div>
        </div>
        <div v-if="initialSubmission.code" class="submission-block">
          <h3 class="block-label">提交的代码</h3>
          <CodeBlock :code="initialSubmission.code" max-height="280px" />
        </div>
      </div>
    </section>

    <!-- 调试记录显示 -->
    <ol v-if="type === 'debug'" class="timeline">
      <li
        v-for="(record, index) in records"
        :key="index"
        :class="['record-detail', record.role]"
      >
        <div class="detail-header">
          <span class="detail-role">
            <AppIcon :name="record.role === 'student' ? 'user' : record.role === 'assistant' ? 'sparkles' : 'file'" :size="15" />
            {{ getRecordRoleLabel(record) }}
          </span>
          <AppBadge size="sm">第 {{ record.round_number }} 轮</AppBadge>
        </div>

        <div class="detail-content">
          <AIResponseDisplay
            v-if="record.role === 'assistant'"
            :ai-response="getRecordAIResponse(record)"
            :student-response="getRecordStudentResponse(record)"
          />
          <div v-else-if="record.role === 'student'" class="student-message plain-text">{{ getStudentContent(record) }}</div>

          <div v-if="record.error" class="detail-error">
            <h4 class="block-label">错误信息</h4>
            <pre class="error-text">{{ record.error }}</pre>
          </div>
        </div>
      </li>
    </ol>

    <!-- 评价记录显示 -->
    <div v-else-if="type === 'evaluate'" class="evaluate-records">
      <section
        v-for="(record, index) in records"
        :key="index"
        :class="['evaluate-record', record.role]"
      >
        <h3 class="section-heading">
          <AppIcon :name="record.role === 'student' ? 'user' : 'sparkles'" :size="16" />
          {{ record.role === 'student' ? '学生提交' : 'AI 评价' }}
        </h3>

        <!-- 学生提交显示代码 -->
        <div v-if="record.role === 'student' && getStudentCode(record)">
          <CodeBlock :code="getStudentCode(record)" max-height="320px" />
        </div>

        <!-- AI评价显示结构化结果 -->
        <template v-if="record.role === 'assistant' && getParsedEvaluation(record)">
          <EvaluationResult
            v-if="hasScores(getParsedEvaluation(record)) || getParsedEvaluation(record).overall_evaluation"
            :result="getParsedEvaluation(record)"
            only-present
          />
          <!-- 如果没有结构化数据，回退到原始JSON -->
          <pre v-else class="payload">{{ formatPayload(record.response_payload) }}</pre>
        </template>
      </section>
    </div>

    <!-- 推荐记录显示 -->
    <div v-else-if="type === 'recommend' && records[0]" class="recommend-record">
      <RecommendationList
        v-if="getParsedRecommendation(records[0])"
        :analysis="getParsedRecommendation(records[0]).analysis || ''"
        :recommendations="getParsedRecommendation(records[0]).recommendations"
      />
      <template v-else>
        <h3 class="section-heading">响应内容</h3>
        <pre class="payload">{{ formatPayload(records[0].response_payload) }}</pre>
      </template>
    </div>
  </AppModal>
</template>

<script setup>
import { ref, computed } from 'vue'
import AIResponseDisplay from '../AIResponseDisplay.vue'
import AppModal from '../ui/AppModal.vue'
import AppIcon from '../ui/AppIcon.vue'
import AppBadge from '../ui/AppBadge.vue'
import CodeBlock from '../code/CodeBlock.vue'
import EvaluationResult from '../results/EvaluationResult.vue'
import RecommendationList from '../results/RecommendationList.vue'

const props = defineProps({
  records: {
    type: Array,
    default: () => []
  },
  initialSubmission: {
    type: Object,
    default: null
  },
  type: {
    type: String,
    required: true,
    validator: (value) => ['debug', 'evaluate', 'recommend'].includes(value)
  }
})

defineEmits(['close'])

// Esc 关闭、焦点陷阱由 AppModal 统一处理
const modalTitle = computed(() => ({
  debug: '调试会话详情',
  evaluate: '代码评价详情',
  recommend: '题目推荐详情'
}[props.type] || '会话详情'))

const showCodeModal = ref(false)

// 格式化载荷内容
const formatPayload = (payload) => {
  if (!payload) return '无'
  try {
    const obj = JSON.parse(payload)
    return JSON.stringify(obj, null, 2)
  } catch (e) {
    return payload
  }
}

// 解析 AI 响应 JSON
const parseAIResponse = (payload) => {
  if (!payload) return null
  try {
    let obj = typeof payload === 'string' ? JSON.parse(payload) : payload
    if (obj.ai_response) {
      return obj.ai_response
    }
    return obj
  } catch (e) {
    return { content: payload }
  }
}

// 获取学生回复内容
const getStudentContent = (record) => {
  if (record.content) return record.content
  if (record.student_response) return record.student_response
  
  if (record.request_payload) {
    try {
      const req = typeof record.request_payload === 'string'
        ? JSON.parse(record.request_payload)
        : record.request_payload
      if (req.student_response) return req.student_response
      return typeof req === 'string' ? req : (req.code ? '提交了代码' : JSON.stringify(req, null, 2))
    } catch {
      return record.request_payload
    }
  }
  return '无'
}

// 获取记录的 role 标签
const getRecordRoleLabel = (record) => {
  if (record.role === 'student') return '你的回复'
  if (record.role === 'assistant') return 'AI 助手'
  return '记录'
}

// 获取记录的 AI 响应
const getRecordAIResponse = (record) => {
  if (record.role === 'assistant') {
    return parseAIResponse(record.response_payload)
  }
  return null
}

// 获取记录的学生回复
const getRecordStudentResponse = (record) => {
  if (record.role === 'student') {
    return getStudentContent(record)
  }
  if (record.role === 'assistant' && record.request_payload) {
    try {
      const req = typeof record.request_payload === 'string'
        ? JSON.parse(record.request_payload)
        : record.request_payload
      return req.student_response || ''
    } catch {
      return ''
    }
  }
  return ''
}

// 获取学生提交的代码
const getStudentCode = (record) => {
  if (!record) return ''
  
  // 优先从 request_payload 中获取
  if (record.request_payload) {
    try {
      const req = typeof record.request_payload === 'string'
        ? JSON.parse(record.request_payload)
        : record.request_payload
      if (req.code) return req.code
    } catch {
      // 忽略解析错误
    }
  }
  
  return ''
}

// 解析评价结果
const getParsedEvaluation = (record) => {
  if (!record || !record.response_payload) return null
  
  try {
    const response = typeof record.response_payload === 'string'
      ? JSON.parse(record.response_payload)
      : record.response_payload
    return response
  } catch {
    return null
  }
}

// 检查是否有评分数据
const hasScores = (evaluation) => {
  if (!evaluation) return false
  return !!(evaluation.functional_correctness ||
            evaluation.logical_rigor ||
            evaluation.algorithm_quality ||
            evaluation.structural_normativity)
}

// 解析推荐结果（结构不符合时返回 null，回退为原始 JSON 展示）
const getParsedRecommendation = (record) => {
  const parsed = getParsedEvaluation(record)
  if (parsed && typeof parsed === 'object' && Array.isArray(parsed.recommendations)) {
    return parsed
  }
  return null
}
</script>

<style scoped>
.block-label {
  margin-bottom: var(--space-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
}

.section-heading {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  font-size: var(--fs-md);
}

.section-heading .app-icon {
  color: var(--color-primary-text);
}

/* 首次提交 */
.initial-submission {
  margin-bottom: var(--space-5);
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.submission-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-3) var(--space-4);
  font-weight: var(--fw-medium);
  color: var(--color-text);
  text-align: left;
  background: var(--color-surface-2);
}

.submission-toggle:hover {
  background: var(--color-surface-hover);
}

.submission-toggle:focus-visible {
  outline-offset: -2px;
}

.submission-title {
  flex: 1;
}

.expand-icon {
  color: var(--color-text-muted);
  transform: rotate(-90deg);
  transition: transform var(--duration) var(--ease);
}

.expand-icon.open {
  transform: rotate(0deg);
}

.submission-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
  border-top: 1px solid var(--color-border);
}

.problem-text {
  max-height: 160px;
  overflow-y: auto;
  padding: var(--space-3);
  font-size: var(--fs-base);
  line-height: 1.65;
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

/* 调试对话时间线 */
.timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.record-detail {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-left: 3px solid var(--color-border-strong);
  border-radius: var(--radius-md);
}

.record-detail.student {
  border-left-color: var(--color-text-muted);
  background: var(--color-surface-2);
}

.record-detail.assistant {
  border-left-color: var(--color-primary);
}

.record-detail.system_error,
.record-detail.ai_service_error {
  border-left-color: var(--color-danger);
  background: var(--color-danger-soft);
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.detail-role {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
}

.student-message {
  font-size: var(--fs-base);
  line-height: 1.7;
}

.detail-error {
  margin-top: var(--space-3);
}

.error-text,
.payload {
  max-height: 320px;
  margin: 0;
  padding: var(--space-3);
  overflow: auto;
  font-size: var(--fs-xs);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--code-text);
  background: var(--code-bg);
  border: 1px solid var(--code-border);
  border-radius: var(--radius-md);
}

.error-text {
  color: var(--color-danger);
}

/* 评价记录 */
.evaluate-records {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}
</style>
