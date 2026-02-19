<template>
  <div class="modal-overlay" @click="$emit('close')">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h2>会话详情</h2>
        <button @click="$emit('close')" class="close-btn">×</button>
      </div>
      
      <div class="modal-body">
        <!-- 首次提交的题目描述和代码（debug/evaluate类型显示） -->
        <div v-if="initialSubmission && type !== 'recommend'" class="initial-submission">
          <div class="submission-header" @click="showCodeModal = !showCodeModal">
            <span class="submission-title">📝 题目描述</span>
            <span class="expand-icon">{{ showCodeModal ? '▼' : '▶' }}</span>
          </div>
          <div v-if="showCodeModal" class="submission-content">
            <div class="problem-description">
              <h4>题目:</h4>
              <div class="problem-text" v-html="initialSubmission.problem_description"></div>
            </div>
            <div class="code-section" v-if="initialSubmission.code">
              <h4>提交的代码:</h4>
              <pre class="code-display">{{ initialSubmission.code }}</pre>
            </div>
          </div>
        </div>
        
        <!-- 调试记录显示 -->
        <template v-if="type === 'debug'">
          <div
            v-for="(record, index) in records"
            :key="index"
            :class="['record-detail', record.role]"
          >
            <div class="detail-header">
              <span class="detail-role">
                {{ getRecordRoleLabel(record) }}
              </span>
              <span class="detail-round">第 {{ record.round_number }} 轮</span>
            </div>
            
            <div class="detail-content">
              <template v-if="record.role === 'assistant'">
                <AIResponseDisplay
                  :ai-response="getRecordAIResponse(record)"
                  :student-response="getRecordStudentResponse(record)"
                />
              </template>
              <template v-else-if="record.role === 'student'">
                <div class="student-message">
                  {{ getStudentContent(record) }}
                </div>
              </template>
              
              <div v-if="record.error" class="detail-error">
                <h4>错误信息:</h4>
                <pre class="error-text">{{ record.error }}</pre>
              </div>
            </div>
          </div>
        </template>
        
        <!-- 评价记录显示 -->
        <template v-else-if="type === 'evaluate'">
          <div
            v-for="(record, index) in records"
            :key="index"
            :class="['record-detail', record.role]"
          >
            <div class="detail-header">
              <span class="detail-role">
                {{ record.role === 'student' ? '👤 学生提交' : '🤖 AI评价' }}
              </span>
            </div>
            
            <div class="detail-content">
              <div v-if="record.role === 'assistant'" class="detail-payload">
                <h4>评价结果:</h4>
                <pre>{{ formatPayload(record.response_payload) }}</pre>
              </div>
            </div>
          </div>
        </template>
        
        <!-- 推荐记录显示 -->
        <template v-else-if="type === 'recommend'">
          <div class="record-detail recommend">
            <div class="detail-header">
              <span class="detail-role">📚 题目推荐</span>
            </div>
            <div class="detail-content" v-if="records[0]">
              <div class="detail-payload">
                <h4>响应内容:</h4>
                <pre>{{ formatPayload(records[0].response_payload) }}</pre>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AIResponseDisplay from '../AIResponseDisplay.vue'

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
  if (record.role === 'student') return '👤 你的回复'
  if (record.role === 'assistant') return '🤖 AI 助手'
  return '📝 记录'
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
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.modal-header h2 {
  font-size: 18px;
  color: #303133;
}

.close-btn {
  background: none;
  border: none;
  font-size: 28px;
  color: #909399;
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.close-btn:hover {
  color: #303133;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.initial-submission {
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 20px;
  overflow: hidden;
}

.submission-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 15px;
  background: #409eff;
  color: white;
  cursor: pointer;
}

.submission-title {
  font-weight: 600;
  font-size: 14px;
}

.expand-icon {
  font-size: 12px;
}

.submission-content {
  padding: 15px;
}

.problem-description {
  margin-bottom: 15px;
}

.problem-description h4,
.code-section h4 {
  font-size: 13px;
  color: #606266;
  margin-bottom: 8px;
}

.problem-text {
  background: white;
  padding: 10px;
  border-radius: 4px;
  font-size: 13px;
  color: #303133;
  max-height: 150px;
  overflow-y: auto;
  white-space: pre-wrap;
  line-height: 1.5;
}

.code-section {
  margin-top: 15px;
}

.code-display {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.record-detail {
  margin-bottom: 20px;
  padding: 15px;
  border-radius: 8px;
  background: #f5f7fa;
}

.record-detail.student {
  background: #f0f9eb;
  border-left: 4px solid #67c23a;
}

.record-detail.assistant {
  background: #ecf5ff;
  border-left: 4px solid #409eff;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 14px;
}

.detail-role {
  font-weight: 500;
}

.detail-round {
  color: #909399;
}

.detail-content h4 {
  font-size: 14px;
  color: #606266;
  margin-bottom: 10px;
}

.detail-payload pre {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 300px;
  overflow-y: auto;
}

.error-text {
  color: #f56c6c;
  background: #2d2d2d;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.student-message {
  background: #f0f9eb;
  padding: 12px;
  border-radius: 6px;
  color: #303133;
  font-size: 14px;
  line-height: 1.6;
}
</style>
