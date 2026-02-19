<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📜 历史记录</h1>
    </div>
    
    <!-- 导航栏 -->
    <div class="history-tabs">
      <button
        :class="['tab-btn', { active: activeTab === 'debug' }]"
        @click="activeTab = 'debug'"
      >
        🤖 AI调试
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'evaluate' }]"
        @click="activeTab = 'evaluate'"
      >
        📝 代码评价
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'recommend' }]"
        @click="activeTab = 'recommend'"
      >
        📚 题目推荐
      </button>
    </div>
    
    <div class="history-content-wrapper">
      <div v-if="loading" class="loading-container">
        <div class="loading"></div>
        <p>加载中...</p>
      </div>
      
      <div v-else-if="errorMessage" class="error-container">
        <p class="message message-error">{{ errorMessage }}</p>
        <button @click="fetchRecords" class="btn btn-primary">重试</button>
      </div>
      
      <div v-else-if="records.length === 0" class="empty-container">
        <div class="empty-icon">📭</div>
        <h3>暂无{{ tabTitle }}记录</h3>
        <p>{{ emptyHint }}</p>
        <router-link :to="emptyLink" class="btn btn-primary">{{ emptyBtnText }}</router-link>
      </div>
      
      <div v-else class="records-list">
        <!-- 调试历史 -->
        <DebugHistoryTab
          v-if="activeTab === 'debug'"
          :records="records"
          @view-details="handleViewDetails"
        />
        
        <!-- 评价历史 -->
        <EvaluateHistoryTab
          v-else-if="activeTab === 'evaluate'"
          :records="records"
          @view-details="handleViewDetails"
        />
        
        <!-- 推荐历史 -->
        <RecommendHistoryTab
          v-else-if="activeTab === 'recommend'"
          :records="records"
          @view-details="handleViewDetails"
        />
      </div>
    </div>
    
    <!-- 详情模态框 - 使用可复用组件 -->
    <HistoryDetailModal
      v-if="showModal"
      :records="selectedRecords"
      :initial-submission="initialSubmission"
      :type="selectedType"
      @close="closeModal"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'
import DebugHistoryTab from '../components/HistoryTabs/DebugHistoryTab.vue'
import EvaluateHistoryTab from '../components/HistoryTabs/EvaluateHistoryTab.vue'
import RecommendHistoryTab from '../components/HistoryTabs/RecommendHistoryTab.vue'
import HistoryDetailModal from '../components/HistoryTabs/HistoryDetailModal.vue'

const authStore = useAuthStore()

// 状态
const activeTab = ref('debug')
const records = ref([])
const loading = ref(false)
const errorMessage = ref('')
const showModal = ref(false)
const selectedRecords = ref([])
const selectedType = ref('debug')
const initialSubmission = ref(null) // 存储首次提交的题目描述和代码
const showCodeModal = ref(false) // 是否显示代码弹窗

// Tab 标题和提示
const tabTitle = computed(() => {
  const titles = {
    debug: '调试',
    evaluate: '评价',
    recommend: '推荐'
  }
  return titles[activeTab.value] || ''
})

const emptyHint = computed(() => {
  const hints = {
    debug: '你还没有与 AI 进行过调试交互',
    evaluate: '你还没有进行过代码评价',
    recommend: '你还没有进行过题目推荐'
  }
  return hints[activeTab.value] || ''
})

const emptyLink = computed(() => {
  const links = {
    debug: '/ai-debug',
    evaluate: '/evaluate',
    recommend: '/recommend'
  }
  return links[activeTab.value] || '/ai-debug'
})

const emptyBtnText = computed(() => {
  const texts = {
    debug: '开始调试',
    evaluate: '开始评价',
    recommend: '开始推荐'
  }
  return texts[activeTab.value] || '开始'
})

// 监听 tab 变化重新获取记录
watch(activeTab, () => {
  fetchRecords()
})

// 获取记录 - 根据 tab 类型获取对应记录
const fetchRecords = async () => {
  loading.value = true
  errorMessage.value = ''
  
  try {
    let response
    switch (activeTab.value) {
      case 'debug':
        response = await aiAPI.getDebugRecords()
        break
      case 'evaluate':
        response = await aiAPI.getEvaluateRecords()
        break
      case 'recommend':
        response = await aiAPI.getRecommendRecords()
        break
      default:
        response = await aiAPI.getRecords()
    }
    if (response && response.data) {
      records.value = response.data
    }
  } catch (error) {
    errorMessage.value = error.error || '获取历史记录失败'
    console.error('Failed to fetch records:', error)
  } finally {
    loading.value = false
  }
}

// 统一处理查看详情事件
const handleViewDetails = ({ records: recs, initialSubmission: initSub, type }) => {
  selectedRecords.value = recs
  initialSubmission.value = initSub
  selectedType.value = type
  showModal.value = true
}

// 关闭模态框
const closeModal = () => {
  showModal.value = false
  selectedRecords.value = []
  initialSubmission.value = null
}

// 格式化日期
const formatDate = (timestamp) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  if (diff < 604800000) return Math.floor(diff / 86400000) + '天前'
  
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

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
    // payload 可能是字符串，需要解析
    let obj = typeof payload === 'string' ? JSON.parse(payload) : payload
    
    // 如果有 ai_response 字段，优先使用
    if (obj.ai_response) {
      return obj.ai_response
    }
    return obj
  } catch (e) {
    // 如果解析失败，返回原始内容作为 content 字段
    return { content: payload }
  }
}

// 获取学生回复内容
const getStudentContent = (record) => {
  // 优先使用 record.content 或 record.student_response
  if (record.content) return record.content
  if (record.student_response) return record.student_response
  
  // 尝试从 request_payload 中获取 student_response
  if (record.request_payload) {
    try {
      // 如果是字符串，解析 JSON
      const req = typeof record.request_payload === 'string'
        ? JSON.parse(record.request_payload)
        : record.request_payload
      
      // 优先返回 student_response
      if (req.student_response) return req.student_response
      
      // 如果没有 student_response，可能是一个学生单独的记录，返回整个 request
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
  // 如果是 AI 回复
  if (record.role === 'assistant') {
    return parseAIResponse(record.response_payload)
  }
  return null
}

// 获取记录的学生回复
const getRecordStudentResponse = (record) => {
  // 如果是学生回复
  if (record.role === 'student') {
    return getStudentContent(record)
  }
  // 如果是 AI 回复，尝试从 request_payload 中获取上一轮的学生回复
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

onMounted(() => {
  if (!authStore.isAuthenticated) {
    window.location.href = '/login'
    return
  }
  fetchRecords()
})
</script>

<style scoped>
.loading-container,
.error-container,
.empty-container {
  max-width: 800px;
  margin: 0 auto;
  text-align: center;
  padding: 60px 20px;
  background: white;
  border-radius: 12px;
}

.loading-container .loading {
  margin-bottom: 15px;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.empty-container h3 {
  color: #303133;
  margin-bottom: 10px;
}

.empty-container p {
  color: #909399;
  margin-bottom: 20px;
}

.records-list {
  max-width: 100%;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.record-group {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.record-group:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.group-info h3 {
  font-size: 16px;
  color: #303133;
  margin-bottom: 5px;
}

.group-time {
  font-size: 12px;
  color: #909399;
}

.btn-sm {
  padding: 6px 12px;
  font-size: 12px;
}

.group-stats {
  display: flex;
  gap: 20px;
}

.stat {
  font-size: 13px;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 5px;
}

.stat-icon {
  font-size: 14px;
}

/* 模态框样式 */
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

/* 首次提交区域样式 */
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

.record-detail.system_error,
.record-detail.ai_service_error {
  background: #fef0f0;
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

.detail-hint {
  font-size: 13px;
  color: #909399;
  margin-bottom: 8px;
}

.student-message {
  background: #f0f9eb;
  padding: 12px;
  border-radius: 6px;
  color: #303133;
  font-size: 14px;
  line-height: 1.6;
}

.history-content-wrapper {
	max-width: 1000px;
	margin: 0 auto;
	padding: 20px;
}
</style>
