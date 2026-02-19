<template>
  <div class="records-list">
    <div
      v-for="group in groupedRecords"
      :key="group.conversation_id"
      class="record-group card"
    >
      <div class="group-header">
        <div class="group-info">
          <h3>评价: {{ group.conversation_id?.substring(0, 15) || 'N/A' }}...</h3>
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
          <span class="stat-icon">📝</span>
          学生提交代码评价
        </span>
        <span class="stat" v-if="group.records.length > 1">
          <span class="stat-icon">💬</span>
          {{ group.records.length }} 条记录
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  records: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['view-details'])

// 按会话分组的记录（评价）
const groupedRecords = computed(() => {
  // 评价类型：按 conversation_id 分组
  const groups = {}
  
  props.records.forEach(record => {
    const convId = record.conversation_id || record.ConversationID
    const createdAt = record.CreatedAt || record.created_at || record.CreatedAt
    
    // 跳过无效记录
    if (!convId) return
    
    if (!groups[convId]) {
      groups[convId] = {
        conversation_id: convId,
        records: [],
        latest_time: new Date(createdAt).getTime()
      }
    }
    groups[convId].records.push(record)
    groups[convId].latest_time = Math.max(
      groups[convId].latest_time,
      new Date(createdAt).getTime()
    )
  })
  
  return Object.values(groups).sort((a, b) => b.latest_time - a.latest_time)
})

// 查看详情
const viewDetails = (group) => {
  const sortedRecords = group.records.sort((a, b) => {
    // student 记录在前，assistant 记录在后
    if (a.role === 'student' && b.role === 'assistant') return -1
    if (a.role === 'assistant' && b.role === 'student') return 1
    return 0
  })
  
  // 提取首次提交的题目描述和代码
  const initialSubmission = extractInitialSubmission(group.records)
  
  emit('view-details', {
    records: sortedRecords,
    initialSubmission,
    type: 'evaluate'
  })
}

// 从记录中提取首次提交的题目描述和代码
const extractInitialSubmission = (records) => {
  // 优先找 round_number 为 1 的学生记录（debug类型）
  let firstRecord = records.find(r => r.round_number === 1 && r.role === 'student')
  
  // 如果没找到，尝试找第一个学生记录（evaluate类型）
  if (!firstRecord) {
    firstRecord = records.find(r => r.role === 'student')
  }
  
  if (firstRecord && firstRecord.request_payload) {
    try {
      const req = typeof firstRecord.request_payload === 'string'
        ? JSON.parse(firstRecord.request_payload)
        : firstRecord.request_payload
      return {
        problem_description: req.problem_description || '',
        code: req.code || ''
      }
    } catch {
      return null
    }
  }
  return null
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
</script>

<style scoped>
.records-list {
  max-width: 1000px;
  margin: 0 auto;
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
</style>