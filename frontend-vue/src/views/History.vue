<template>
  <div class="history-page">
    <PageHeader
      title="历史记录"
      description="回看你的 AI 调试对话、代码评价与题目推荐。"
    />

    <AppTabs v-model="activeTab" :tabs="tabs" label="历史记录类型">
      <AppSpinner v-if="loading" block label="加载中..." />

      <div v-else-if="errorMessage" class="state-card">
        <AppEmpty icon="alert" tone="danger" title="加载失败" :description="errorMessage">
          <template #actions>
            <AppButton variant="primary" icon="refresh" @click="fetchRecords">重试</AppButton>
          </template>
        </AppEmpty>
      </div>

      <div v-else-if="records.length === 0" class="state-card">
        <AppEmpty icon="inbox" :title="`暂无${tabTitle}记录`" :description="emptyHint">
          <template #actions>
            <AppButton variant="primary" :to="emptyLink">{{ emptyBtnText }}</AppButton>
          </template>
        </AppEmpty>
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
    </AppTabs>

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
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'
import DebugHistoryTab from '../components/HistoryTabs/DebugHistoryTab.vue'
import EvaluateHistoryTab from '../components/HistoryTabs/EvaluateHistoryTab.vue'
import RecommendHistoryTab from '../components/HistoryTabs/RecommendHistoryTab.vue'
import HistoryDetailModal from '../components/HistoryTabs/HistoryDetailModal.vue'
import PageHeader from '../components/ui/PageHeader.vue'
import AppTabs from '../components/ui/AppTabs.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppSpinner from '../components/ui/AppSpinner.vue'

const tabs = [
  { key: 'debug', label: 'AI 调试', icon: 'chat', shortcut: 'Ctrl+1' },
  { key: 'evaluate', label: '代码评价', icon: 'evaluate', shortcut: 'Ctrl+2' },
  { key: 'recommend', label: '题目推荐', icon: 'target', shortcut: 'Ctrl+3' }
]

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

// 标签页切换快捷键
const tabKeys = ['ctrl+1', 'ctrl+2', 'ctrl+3']
useKeyboardShortcut(tabKeys, (event) => {
  const index = parseInt(event.key) - 1
  if (index >= 0 && index < 3) {
    const tabMap = ['debug', 'evaluate', 'recommend']
    activeTab.value = tabMap[index]
  }
})
</script>

<style scoped>
.history-page {
  max-width: 960px;
  margin: 0 auto;
}

.state-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
</style>
