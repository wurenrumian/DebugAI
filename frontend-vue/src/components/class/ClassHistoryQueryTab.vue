<template>
  <div class="class-history-query-tab">
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
          <span>至</span>
          <input type="date" v-model="endDate" class="date-input" />
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary" @click="handleQuery" :disabled="classStore.loading">
          {{ classStore.loading ? '查询中...' : '查询' }}
        </button>
        <button class="btn btn-secondary" @click="handleExport" :disabled="classStore.loading">
          导出
        </button>
      </div>
    </div>

    <!-- 历史记录标签页 -->
    <div class="history-tabs">
      <div class="tab-buttons">
        <button 
          :class="['tab-btn', { active: activeType === 'debug' }]" 
          @click="activeType = 'debug'"
        >
          Debug 记录
        </button>
        <button 
          :class="['tab-btn', { active: activeType === 'evaluate' }]" 
          @click="activeType = 'evaluate'"
        >
          Evaluate 记录
        </button>
        <button 
          :class="['tab-btn', { active: activeType === 'recommend' }]" 
          @click="activeType = 'recommend'"
        >
          Recommend 记录
        </button>
      </div>

      <!-- 标签页内容 -->
      <div class="tab-content">
        <!-- Debug 记录 -->
        <div v-show="activeType === 'debug'" class="tab-pane">
          <DebugHistoryTab 
            :records="debugRecords" 
            @view-details="handleViewDetails"
          />
        </div>

        <!-- Evaluate 记录 -->
        <div v-show="activeType === 'evaluate'" class="tab-pane">
          <EvaluateHistoryTab 
            :records="evaluateRecords" 
            @view-details="handleViewDetails"
          />
        </div>

        <!-- Recommend 记录 -->
        <div v-show="activeType === 'recommend'" class="tab-pane">
          <RecommendHistoryTab 
            :records="recommendRecords" 
            @view-details="handleViewDetails"
          />
        </div>
      </div>
    </div>

    <!-- 详情弹窗 -->
    <HistoryDetailModal
      v-if="showModal"
      :records="selectedRecords"
      :initial-submission="initialSubmission"
      :type="selectedType"
      @close="showModal = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useClassStore } from '../../stores/class'
import DebugHistoryTab from '../HistoryTabs/DebugHistoryTab.vue'
import EvaluateHistoryTab from '../HistoryTabs/EvaluateHistoryTab.vue'
import RecommendHistoryTab from '../HistoryTabs/RecommendHistoryTab.vue'
import HistoryDetailModal from '../HistoryTabs/HistoryDetailModal.vue'

const classStore = useClassStore()

// 筛选条件
const selectedStudents = ref([])
const startDate = ref('')
const endDate = ref('')
const activeType = ref('debug')

// 详情弹窗
const showModal = ref(false)
const selectedRecords = ref([])
const initialSubmission = ref(null)
const selectedType = ref('')

// 从班级成员中筛选学生
const students = computed(() => {
  return classStore.members.filter(m => m.role === 'student') || []
})

// 获取查询参数
function getQueryParams() {
  const params = {}
  if (selectedStudents.value.length > 0) {
    params.student_ids = selectedStudents.value.join(',')
  }
  if (startDate.value) {
    params.start_date = startDate.value
  }
  if (endDate.value) {
    params.end_date = endDate.value
  }
  return params
}

// 处理查询
async function handleQuery() {
  if (!classStore.currentClass) return
  
  const classId = classStore.currentClass.id
  const params = getQueryParams()
  
  await Promise.all([
    classStore.fetchClassDebugRecords(classId, params),
    classStore.fetchClassEvaluateRecords(classId, params),
    classStore.fetchClassRecommendRecords(classId, params)
  ])
}

// 处理导出
async function handleExport() {
  if (!classStore.currentClass) return
  
  const classId = classStore.currentClass.id
  const params = getQueryParams()
  
  if (activeType.value === 'debug') {
    await classStore.exportClassDebugRecords(classId, params)
  } else if (activeType.value === 'evaluate') {
    await classStore.exportClassEvaluateRecords(classId, params)
  } else if (activeType.value === 'recommend') {
    await classStore.exportClassRecommendRecords(classId, params)
  }
}

// 按 conversation_id 分组
function groupByConversation(records) {
  if (!records || !Array.isArray(records)) return []
  
  const groups = {}
  records.forEach(r => {
    const convId = r.conversation_id
    if (!groups[convId]) {
      groups[convId] = { 
        conversation_id: convId, 
        records: [], 
        latest_time: r.created_at,
        max_round: r.round_number || 1
      }
    }
    groups[convId].records.push(r)
    if (r.round_number && r.round_number > groups[convId].max_round) {
      groups[convId].max_round = r.round_number
    }
    if (new Date(r.created_at) > new Date(groups[convId].latest_time)) {
      groups[convId].latest_time = r.created_at
    }
  })
  return Object.values(groups).sort((a, b) => 
    new Date(b.latest_time) - new Date(a.latest_time)
  )
}

// 计算属性：转换后的记录
const debugRecords = computed(() => {
  return groupByConversation(classStore.classDebugRecords.data)
})

const evaluateRecords = computed(() => {
  return groupByConversation(classStore.classEvaluateRecords.data)
})

const recommendRecords = computed(() => {
  return groupByConversation(classStore.classRecommendRecords.data)
})

// 查看详情
function handleViewDetails({ records, initialSubmission: initSub, type }) {
  selectedRecords.value = records
  initialSubmission.value = initSub
  selectedType.value = type
  showModal.value = true
}

// 页面加载时自动查询
onMounted(() => {
  handleQuery()
})
</script>

<style scoped>
.class-history-query-tab {
  padding: 16px;
}

.filter-section {
  background: #f5f5f5;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 16px;
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
  font-weight: 500;
  white-space: nowrap;
}

.multi-select {
  min-width: 200px;
  height: 100px;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.date-input {
  padding: 4px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.hint {
  font-size: 12px;
  color: #666;
}

.filter-actions {
  display: flex;
  gap: 12px;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: #007bff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #0056b3;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover:not(:disabled) {
  background: #545b62;
}

.history-tabs {
  background: white;
  border-radius: 8px;
}

.tab-buttons {
  display: flex;
  border-bottom: 1px solid #ddd;
}

.tab-btn {
  padding: 12px 24px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: #666;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #007bff;
}

.tab-btn.active {
  color: #007bff;
  border-bottom-color: #007bff;
}

.tab-content {
  padding: 16px;
}

.tab-pane {
  min-height: 300px;
}

.loading {
  text-align: center;
  padding: 40px;
  color: #666;
}
</style>
