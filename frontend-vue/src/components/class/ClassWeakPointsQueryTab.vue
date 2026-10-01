<template>
  <div class="class-weak-points-query-tab">
    <!-- 筛选器区域 -->
    <ClassRecordFilters
      v-model:selected-students="selectedStudents"
      v-model:start-date="startDate"
      v-model:end-date="endDate"
      :students="students"
    >
      <template #actions>
        <AppButton variant="primary" icon="search" :loading="classStore.loading" @click="handleQuery">
          {{ classStore.loading ? '查询中...' : '查询' }}
        </AppButton>
        <AppButton icon="download" :disabled="classStore.loading || !classStore.currentClass" @click="handleExport">
          导出CSV
        </AppButton>
      </template>
    </ClassRecordFilters>

    <!-- 结果展示区域 -->
    <div class="result-section">
      <AppSpinner v-if="classStore.loading" block label="加载中..." />

      <div v-else-if="classStore.classWeakPoints.length === 0" class="state-card">
        <AppEmpty
          icon="chart"
          title="暂无薄弱点数据"
          description="所选时间范围内没有统计到薄弱点，可以调整筛选条件后重新查询。"
          compact
        />
      </div>

      <div v-else class="weak-points-result">
        <!-- 统计信息 -->
        <div class="stats-bar">
          <div class="stat-tile">
            <span class="stat-label">学生人数</span>
            <span class="stat-value">{{ studentCount }}</span>
          </div>
          <div class="stat-tile">
            <span class="stat-label">薄弱点总次数</span>
            <span class="stat-value">{{ totalWeakPoints }}</span>
          </div>
          <div class="stat-tile">
            <span class="stat-label">涉及关键词</span>
            <span class="stat-value">{{ transformedWeakPoints.length }}</span>
          </div>
        </div>

        <!-- 薄弱点展示 -->
        <WeakPointDisplay
          :weak-points="transformedWeakPoints"
          :show-description="true"
          :show-charts="true"
          chart-position="top"
          :top-n="15"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useClassStore } from '../../stores/class'
import { classAPI } from '../../api'
import WeakPointDisplay from '../WeakPointDisplay.vue'
import { toast } from '../../utils/toast'
import { getErrorMessage } from '../../utils/errors'
import ClassRecordFilters from './ClassRecordFilters.vue'
import AppButton from '../ui/AppButton.vue'
import AppEmpty from '../ui/AppEmpty.vue'
import AppSpinner from '../ui/AppSpinner.vue'

const classStore = useClassStore()

// 获取今天的日期字符串
function getTodayString() {
  const today = new Date()
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 筛选条件
const selectedStudents = ref([])
const startDate = ref(getTodayString())
const endDate = ref(getTodayString())

// 从班级成员中筛选学生
const students = computed(() => {
  return classStore.members.filter(m => m.role === 'student') || []
})

// 统计信息
const studentCount = computed(() => {
  // 如果后端返回的是直接的薄弱点列表，则学生人数可能无法直接从该列表获取
  // 这里假设如果数据结构是直接的列表，我们可能需要从 classStore.members 获取或者根据业务逻辑调整
  // 如果 classStore.classWeakPoints 已经是薄弱点列表，那么 length 就是薄弱点种类的数量
  return classStore.members.filter(m => m.role === 'student').length
})

const totalWeakPoints = computed(() => {
  if (classStore.classWeakPoints.length > 0 && !classStore.classWeakPoints[0].weak_points) {
    // 直接列表格式
    return classStore.classWeakPoints.reduce((sum, wp) => sum + (wp.count || 0), 0)
  }
  // 按学生聚合格式
  return classStore.classWeakPoints.reduce((sum, student) => {
    return sum + (student.weak_points?.reduce((s, wp) => s + wp.count, 0) || 0)
  }, 0)
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

// 转换数据：兼容后端返回的两种格式
// 1. 按学生聚合格式: [{ student_id: 1, weak_points: [...] }, ...]
// 2. 直接薄弱点列表格式: [{ category: "...", count: 1, ... }, ...]
const transformedWeakPoints = computed(() => {
  if (classStore.classWeakPoints.length === 0) return []

  // 检查是否是直接的薄弱点列表格式 (如用户提供的 JSON 示例)
  if (!classStore.classWeakPoints[0].weak_points) {
    return [...classStore.classWeakPoints].sort((a, b) => b.count - a.count)
  }

  // 按学生聚合格式转换为按关键词聚合
  const keywordMap = {}
  classStore.classWeakPoints.forEach(student => {
    if (student.weak_points && Array.isArray(student.weak_points)) {
      student.weak_points.forEach(wp => {
        const key = wp.keyword
        if (!keywordMap[key]) {
          keywordMap[key] = {
            keyword: wp.keyword,
            category: wp.category,
            count: 0,
            description: wp.description || ''
          }
        }
        keywordMap[key].count += wp.count
      })
    }
  })
  
  return Object.values(keywordMap).sort((a, b) => b.count - a.count)
})

// 处理查询
async function handleQuery() {
  if (!classStore.currentClass) return
  
  const classId = classStore.currentClass.id
  const params = getQueryParams()
  
  await classStore.fetchClassWeakPoints(classId, params)
}

// 处理导出 - 调用后端API导出CSV
async function handleExport() {
  if (!classStore.currentClass) return
  
  try {
    const classId = classStore.currentClass.id
    const params = getQueryParams()
    
    // 注意：响应拦截器返回 response.data，所以这里直接用 response
    const blob = await classAPI.exportClassWeakPointsCSV(classId, params)
    
    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([blob]))
    const link = document.createElement('a')
    link.href = url
    link.download = `weak_points_class_${classId}_${new Date().toISOString().split('T')[0]}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('导出失败:', error)
    toast.error(getErrorMessage(error, '导出失败，请重试'))
  }
}

// 页面加载时自动查询
onMounted(() => {
  handleQuery()
})
</script>

<style scoped>
.state-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.stats-bar {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.stat-tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.stat-label {
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

.stat-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-semibold);
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 640px) {
  .stats-bar {
    gap: var(--space-2);
  }

  .stat-tile {
    padding: var(--space-3);
  }

  .stat-value {
    font-size: var(--fs-xl);
  }
}
</style>
