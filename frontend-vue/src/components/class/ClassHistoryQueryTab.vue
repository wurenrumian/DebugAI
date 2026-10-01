<template>
  <div class="class-history-tab">
    <!-- 筛选器区域 -->
    <ClassRecordFilters
      v-model:selected-students="selectedStudents"
      v-model:start-date="startDate"
      v-model:end-date="endDate"
      :students="students"
    >
      <template #actions>
        <AppButton variant="primary" icon="search" :loading="loading" @click="handleQuery">
          {{ loading ? '查询中...' : '查询' }}
        </AppButton>
      </template>
    </ClassRecordFilters>

    <!-- 记录类型 -->
    <div class="type-switch" role="group" aria-label="记录类型">
      <button
        v-for="opt in typeOptions"
        :key="opt.key"
        type="button"
        :aria-pressed="activeTab === opt.key ? 'true' : 'false'"
        :class="['type-option', { active: activeTab === opt.key }]"
        @click="activeTab = opt.key"
      >
        <AppIcon :name="opt.icon" :size="15" />
        {{ opt.label }}
      </button>
    </div>

    <div class="history-content">
      <AppSpinner v-if="loading" block label="加载中..." />

      <div v-else-if="errorMessage" class="state-card">
        <AppEmpty icon="alert" tone="danger" title="查询失败" :description="errorMessage" compact>
          <template #actions>
            <AppButton variant="primary" icon="refresh" @click="handleQuery">重试</AppButton>
          </template>
        </AppEmpty>
      </div>

      <div v-else-if="records.length === 0" class="state-card">
        <AppEmpty icon="inbox" :title="`暂无${tabTitle}记录`" :description="emptyHint" compact />
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
import ClassRecordFilters from './ClassRecordFilters.vue'
import AppButton from '../ui/AppButton.vue'
import AppEmpty from '../ui/AppEmpty.vue'
import AppIcon from '../ui/AppIcon.vue'
import AppSpinner from '../ui/AppSpinner.vue'

const typeOptions = [
  { key: 'debug', label: 'AI 调试', icon: 'chat' },
  { key: 'evaluate', label: '代码评价', icon: 'evaluate' },
  { key: 'recommend', label: '题目推荐', icon: 'target' }
]

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
.type-switch {
  display: inline-flex;
  gap: 2px;
  max-width: 100%;
  padding: 3px;
  margin-bottom: var(--space-4);
  overflow-x: auto;
  background: var(--color-surface-3);
  border-radius: var(--radius-md);
}

.type-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 6px 14px;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  transition: background-color var(--duration-fast) var(--ease), color var(--duration-fast) var(--ease);
}

.type-option:hover {
  color: var(--color-text);
}

.type-option.active {
  color: var(--color-text);
  background: var(--color-surface);
  box-shadow: var(--shadow-sm);
}

.state-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
</style>
