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
          <span class="date-separator">至</span>
          <input type="date" v-model="endDate" class="date-input" />
        </div>
      </div>
      <div class="filter-actions">
        <button class="btn btn-primary" @click="handleQuery" :disabled="classStore.loading">
          {{ classStore.loading ? '查询中...' : '查询' }}
        </button>
        <button class="btn btn-secondary" @click="handleExport" :disabled="classStore.loading || !classStore.currentClass">
          导出CSV
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
        
        <!-- 薄弱点展示 -->
        </div>
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
    alert('导出失败，请重试')
  }
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
  border-color: #409eff;
}

.date-input {
  padding: 8px 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.3s;
}

.date-input:focus {
  border-color: #409eff;
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

.btn-secondary {
  background: #f5f7fa;
  color: #606266;
  border: 1px solid #dcdfe6;
}

.btn-secondary:hover:not(:disabled) {
  background: #ecf5ff;
  border-color: #409eff;
  color: #409eff;
}

.result-section {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  padding: 16px;
  min-height: 300px;
}

.stats-bar {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 6px;
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
  color: #909399;
}

.weak-points-result {
  margin-top: 16px;
}
</style>
