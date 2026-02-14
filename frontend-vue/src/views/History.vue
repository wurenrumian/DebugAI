<template>
  <div class="history-container">
    <div class="history-header">
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
    
    <div class="history-content">
      <div v-if="loading" class="loading-container">
        <div class="loading"></div>
        <p>加载中...</p>
      </div>
      
      <div v-else-if="errorMessage" class="error-container">
        <p class="message message-error">{{ errorMessage }}</p>
        <button @click="fetchRecords" class="btn btn-primary">重试</button>
      </div>
      
      <div v-else-if="displayRecords.length === 0" class="empty-container">
        <div class="empty-icon">📭</div>
        <h3>暂无{{ tabTitle }}记录</h3>
        <p>{{ emptyHint }}</p>
        <router-link :to="emptyLink" class="btn btn-primary">{{ emptyBtnText }}</router-link>
      </div>
      
      <div v-else class="records-list">
        <!-- 调试历史 -->
        <template v-if="activeTab === 'debug'">
          <div
            v-for="group in groupedRecords"
            :key="group.conversation_id"
            class="record-group card"
          >
            <div class="group-header">
              <div class="group-info">
                <h3>会话: {{ group.conversation_id.substring(0, 15) }}...</h3>
                <span class="group-time">{{ formatDate(group.latest_time) }}</span>
              </div>
              <button
                @click="viewDetails(group)"
                class="btn btn-secondary btn-sm"
              >
                查看详情
              </button>
            </div>
            
            <div class="group-stats">
              <span class="stat">
                <span class="stat-icon">💬</span>
                {{ group.records.length }} 条记录
              </span>
              <span class="stat">
                <span class="stat-icon">🔄</span>
                轮次: {{ group.max_round }}
              </span>
            </div>
          </div>
        </template>
        
        <!-- 评价历史 -->
        <template v-else-if="activeTab === 'evaluate'">
          <div
            v-for="record in displayRecords"
            :key="record.id"
            class="record-group card"
          >
            <div class="group-header">
              <div class="group-info">
                <h3>评价: {{ record.conversation_id?.substring(0, 15) || 'N/A' }}...</h3>
                <span class="group-time">{{ formatDate(record.created_at) }}</span>
              </div>
              <button
                @click="viewEvaluateDetails(record)"
                class="btn btn-secondary btn-sm"
              >
                查看详情
              </button>
            </div>
            
            <div class="group-stats">
              <span class="stat">
                <span class="stat-icon">📝</span>
                代码评价
              </span>
            </div>
          </div>
        </template>
        
        <!-- 推荐历史 -->
        <template v-else-if="activeTab === 'recommend'">
          <div
            v-for="record in displayRecords"
            :key="record.id"
            class="record-group card"
          >
            <div class="group-header">
              <div class="group-info">
                <h3>推荐记录</h3>
                <span class="group-time">{{ formatDate(record.created_at) }}</span>
              </div>
              <button
                @click="viewRecommendDetails(record)"
                class="btn btn-secondary btn-sm"
              >
                查看详情
              </button>
            </div>
            
            <div class="group-stats">
              <span class="stat">
                <span class="stat-icon">📚</span>
                题目推荐
              </span>
            </div>
          </div>
        </template>
      </div>
    </div>
    
    <!-- 详情模态框 -->
    <div v-if="showModal" class="modal-overlay" @click="closeModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h2>会话详情</h2>
          <button @click="closeModal" class="close-btn">×</button>
        </div>
        
        <div class="modal-body">
          <div 
            v-for="(record, index) in selectedRecords" 
            :key="index" 
            :class="['record-detail', record.role]"
          >
            <div class="detail-header">
              <span class="detail-role">
                {{ record.role === 'student' ? '👤 学生' : '🤖 AI 助手' }}
              </span>
              <span class="detail-round">第 {{ record.round_number }} 轮</span>
            </div>
            
            <div class="detail-content">
              <div v-if="record.role === 'student'" class="detail-payload">
                <h4>请求内容:</h4>
                <pre>{{ formatPayload(record.request_payload) }}</pre>
              </div>
              
              <div v-else-if="record.role === 'assistant'" class="detail-payload">
                <h4>响应内容:</h4>
                <pre>{{ formatPayload(record.response_payload) }}</pre>
              </div>
              
              <div v-else class="detail-error">
                <h4>错误信息:</h4>
                <pre class="error-text">{{ record.error }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'

const authStore = useAuthStore()

// 状态
const activeTab = ref('debug')
const records = ref([])
const loading = ref(false)
const errorMessage = ref('')
const showModal = ref(false)
const selectedRecords = ref([])

// 计算属性：根据 tab 筛选记录
const displayRecords = computed(() => {
  if (activeTab.value === 'debug') {
    // 调试记录：round_number > 0
    return records.value.filter(r => r.round_number > 0)
  } else if (activeTab.value === 'evaluate') {
    // 评价记录：conversation_id 以 eval_ 开头
    return records.value.filter(r => r.conversation_id?.startsWith('eval_'))
  } else if (activeTab.value === 'recommend') {
    // 推荐记录：TODO - 需要后端支持
    return []
  }
  return records.value
})

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

// 按会话分组的记录
const groupedRecords = computed(() => {
  const debugRecords = records.value.filter(r => r.round_number > 0)
  const groups = {}
  
  debugRecords.forEach(record => {
    const convId = record.conversation_id
    if (!groups[convId]) {
      groups[convId] = {
        conversation_id: convId,
        records: [],
        latest_time: new Date(record.created_at).getTime(),
        max_round: 0
      }
    }
    groups[convId].records.push(record)
    groups[convId].max_round = Math.max(groups[convId].max_round, record.round_number)
    groups[convId].latest_time = Math.max(
      groups[convId].latest_time,
      new Date(record.created_at).getTime()
    )
  })
  
  return Object.values(groups).sort((a, b) => b.latest_time - a.latest_time)
})

// 获取记录
const fetchRecords = async () => {
  loading.value = true
  errorMessage.value = ''
  
  try {
    const response = await aiAPI.getRecords()
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

// 查看评价详情
const viewEvaluateDetails = (record) => {
  selectedRecords.value = [record]
  showModal.value = true
}

// 查看推荐详情
const viewRecommendDetails = (record) => {
  selectedRecords.value = [record]
  showModal.value = true
}

// 查看详情
const viewDetails = (group) => {
  selectedRecords.value = group.records.sort((a, b) => a.round_number - b.round_number)
  showModal.value = true
}

// 关闭模态框
const closeModal = () => {
  showModal.value = false
  selectedRecords.value = []
}

// 格式化日期
const formatDate = (timestamp) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  
  // 小于1分钟
  if (diff < 60000) {
    return '刚刚'
  }
  // 小于1小时
  if (diff < 3600000) {
    return Math.floor(diff / 60000) + '分钟前'
  }
  // 小于1天
  if (diff < 86400000) {
    return Math.floor(diff / 3600000) + '小时前'
  }
  // 小于7天
  if (diff < 604800000) {
    return Math.floor(diff / 86400000) + '天前'
  }
  
  // 超过7天，显示具体日期
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

onMounted(() => {
  if (!authStore.isAuthenticated) {
    window.location.href = '/login'
    return
  }
  fetchRecords()
})
</script>

<style scoped>
.history-container {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.history-header {
  background: white;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.history-header h1 {
  font-size: 24px;
  color: #303133;
  margin-top: 10px;
}

.back-link {
  color: #409eff;
  font-size: 14px;
}

.back-link:hover {
  text-decoration: underline;
}

/* Tab 导航 */
.history-tabs {
  max-width: 1200px;
  margin: 0 auto 20px;
  display: flex;
  gap: 10px;
  padding: 0 20px;
}

.tab-btn {
  padding: 12px 24px;
  background: white;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  color: #666;
  cursor: pointer;
  transition: all 0.3s;
}

.tab-btn:hover {
  border-color: #667eea;
  color: #667eea;
}

.tab-btn.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-color: transparent;
  color: white;
}

.history-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.loading-container,
.error-container,
.empty-container {
  text-align: center;
  padding: 60px 20px;
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
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.record-group {
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

.record-detail {
  margin-bottom: 20px;
  padding: 15px;
  border-radius: 8px;
  background: #f5f7fa;
}

.record-detail.student {
  background: #ecf5ff;
}

.record-detail.assistant {
  background: #f0f9eb;
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
</style>
