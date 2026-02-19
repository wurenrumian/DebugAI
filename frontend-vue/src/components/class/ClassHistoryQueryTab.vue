<template>
  <div class="page-container">
    <!-- 筛选器区域 -->
    <div class="filter-section">
      <div class="filter-row">
        <div class="filter-item">
          <label>学生筛选：</label>
          <select v-model="selectedStudents" multiple class="multi-select">
            <option v-for="student in students" :key="student.student_id" :value="student.student_id">
              {{ student.username }} ({{ student.student_id }})
            </option>
          </select>
          <span class="hint">按住 Ctrl/Cmd 可多选，不选默认为全班</span>
        </div>
        <div class="filter-item">
          <label>时间范围：</label>
          <input type="date" v-model="startDate" class="date-input" />
          <span class="date-separator">至</span>
          <input type="date" v-model="endDate" class="date-input" />
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary" @click="handleQuery" :disabled="loading">
          {{ loading ? '查询中...' : '查询' }}
        </button>
      </div>
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
        <button @click="handleQuery" class="btn btn-primary">重试</button>
      </div>
      
      <div v-else-if="records.length === 0" class="empty-container">
        <div class="empty-icon">📭</div>
        <h3>暂无{{ tabTitle }}记录</h3>
        <p>{{ emptyHint }}</p>
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
    
    <!-- 详情模态框 -->
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
import { useClassStore } from '../../stores/class'
import { classAPI } from '../../api'
import DebugHistoryTab from '../HistoryTabs/DebugHistoryTab.vue'
import EvaluateHistoryTab from '../HistoryTabs/EvaluateHistoryTab.vue'
import RecommendHistoryTab from '../HistoryTabs/RecommendHistoryTab.vue'
import HistoryDetailModal from '../HistoryTabs/HistoryDetailModal.vue'

const classStore = useClassStore()

// 获取今天的日期字符串
function getTodayString() {
  const today = new Date()
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 状态 - 使用与 History.vue 相同的方式
const activeTab = ref('debug')
const records = ref([])
const loading = ref(false)
const errorMessage = ref('')
const showModal = ref(false)
const selectedRecords = ref([])
const selectedType = ref('debug')
const initialSubmission = ref(null)

// 筛选条件
const selectedStudents = ref([])
const startDate = ref(getTodayString())
const endDate = ref(getTodayString())

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
    debug: '该班级暂无调试记录',
    evaluate: '该班级暂无评价记录',
    recommend: '该班级暂无推荐记录'
  }
  return hints[activeTab.value] || ''
})

// 从班级成员中筛选学生
const students = computed(() => {
  return classStore.members.filter(m => m.role === 'student') || []
})

// 获取查询参数
function getQueryParams() {
  const params = {}
  if (selectedStudents.value.length > 0) {
    // 发送 JSON 数组格式，符合后端期望
    params.student_ids = JSON.stringify(selectedStudents.value)
  }
  if (startDate.value) {
    params.start_date = startDate.value
  }
  if (endDate.value) {
    params.end_date = endDate.value
  }
  return params
}

// 处理查询 - 使用与 History.vue 相同的方式
async function handleQuery() {
  if (!classStore.currentClass) return

  loading.value = true
  errorMessage.value = ''
  
  const classId = classStore.currentClass.id
  const params = getQueryParams()

  try {
    let response
    switch (activeTab.value) {
      case 'debug':
        response = await classAPI.getClassDebugRecords(classId, params)
        break
      case 'evaluate':
        response = await classAPI.getClassEvaluateRecords(classId, params)
        break
      case 'recommend':
        response = await classAPI.getClassRecommendRecords(classId, params)
        break
      default:
        response = await classAPI.getClassDebugRecords(classId, params)
    }
    // 直接设置 records，与 History.vue 相同的方式
    if (response && response.data) {
      records.value = response.data
    }
  } catch (error) {
    errorMessage.value = error.error || '查询失败'
    console.error('Failed to fetch records:', error)
  } finally {
    loading.value = false
  }
}

// 查看详情
function handleViewDetails({ records: recs, initialSubmission: initSub, type }) {
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

// 监听 activeTab 变化，自动重新查询
watch(activeTab, () => {
  handleQuery()
})

// 监听班级变化，当班级切换时重新查询
watch(() => classStore.currentClass, (newClass) => {
  if (newClass) {
    handleQuery()
  }
}, { deep: true })

// 页面加载时自动查询
onMounted(() => {
  handleQuery()
})
</script>

<style scoped>
.page-container {
  padding: 16px;
}

.filter-section {
  background: white;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.filter-row {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-item label {
  font-weight: 600;
  white-space: nowrap;
  color: #303133;
  font-size: 14px;
}

.multi-select {
  min-width: 200px;
  height: 100px;
  padding: 8px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  background: white;
  transition: border-color 0.3s;
}

.multi-select:focus {
  border-color: #667eea;
}

.date-input {
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.3s;
}

.date-input:focus {
  border-color: #667eea;
}

.date-separator {
  color: #909399;
  font-size: 14px;
}

.hint {
  font-size: 12px;
  color: #909399;
}

.filter-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #409eff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #66b1ff;
}

/* 历史记录标签页样式 - 与 History.vue 相同 */
.history-tabs {
  display: flex;
  border-bottom: 1px solid #ebeef5;
  background: #f5f7fa;
  padding: 0 12px;
}

.tab-btn {
  padding: 12px 20px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #409eff;
  background: white;
}

.tab-btn.active {
  color: #409eff;
  background: white;
  border-bottom-color: #409eff;
}

.history-content-wrapper {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

/* 加载、错误、空状态样式 - 与 History.vue 相同 */
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
}

.message-error {
  color: #f56c6c;
  margin-bottom: 15px;
}

.records-list {
  max-width: 100%;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 15px;
}
</style>
