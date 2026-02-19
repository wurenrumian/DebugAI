<template>
  <div class="class-weak-points-query-tab">
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
        <button class="btn btn-secondary" @click="handleExport" :disabled="classStore.loading || !classStore.classWeakPoints.length">
          导出
        </button>
      </div>
    </div>

    <!-- 结果展示区域 -->
    <div class="result-section">
      <div v-if="classStore.loading" class="loading">
        加载中...
      </div>
      <div v-else-if="classStore.classWeakPoints.length === 0" class="empty-state">
        暂无薄弱点数据
      </div>
      <div v-else class="weak-points-result">
        <!-- 统计信息 -->
        <div class="stats-bar">
          <span class="stat-item">
            <span class="stat-icon">👥</span>
            {{ studentCount }} 位学生
          </span>
          <span class="stat-item">
            <span class="stat-icon">🎯</span>
            {{ totalWeakPoints }} 个薄弱点
          </span>
        </div>
        
        <!-- 薄弱点展示 -->
        <WeakPointDisplay 
          :weak-points="transformedWeakPoints"
          :show-description="true"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useClassStore } from '../../stores/class'
import WeakPointDisplay from '../WeakPointDisplay.vue'

const classStore = useClassStore()

// 筛选条件
const selectedStudents = ref([])
const startDate = ref('')
const endDate = ref('')

// 从班级成员中筛选学生
const students = computed(() => {
  return classStore.members.filter(m => m.role === 'student') || []
})

// 统计信息
const studentCount = computed(() => {
  return classStore.classWeakPoints.length
})

const totalWeakPoints = computed(() => {
  return classStore.classWeakPoints.reduce((sum, student) => {
    return sum + (student.weak_points?.reduce((s, wp) => s + wp.count, 0) || 0)
  }, 0)
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

// 转换数据：将后端返回的"按学生聚合"格式转换为"按关键词聚合"格式
const transformedWeakPoints = computed(() => {
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

// 处理导出
function handleExport() {
  if (!classStore.classWeakPoints.length) return
  
  // 导出转换后的数据
  const exportData = {
    export_time: new Date().toISOString(),
    class_id: classStore.currentClass?.id,
    class_name: classStore.currentClass?.class_name,
    student_count: studentCount.value,
    total_weak_points: totalWeakPoints.value,
    weak_points: transformedWeakPoints.value
  }
  
  const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `weak_points_${classStore.currentClass?.id}.json`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// 页面加载时自动查询
onMounted(() => {
  handleQuery()
})
</script>

<style scoped>
.class-weak-points-query-tab {
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

.result-section {
  background: white;
  border-radius: 8px;
  padding: 16px;
  min-height: 300px;
}

.stats-bar {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: #f8f9fa;
  border-radius: 4px;
  margin-bottom: 16px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: #495057;
}

.stat-icon {
  font-size: 16px;
}

.loading, .empty-state {
  text-align: center;
  padding: 40px;
  color: #666;
}

.weak-points-result {
  margin-top: 16px;
}
</style>
