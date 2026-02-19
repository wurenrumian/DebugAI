<template>
  <div class="records-list">
    <div
      v-for="record in records"
      :key="record.ID"
      class="record-group card"
    >
      <div class="group-header">
        <div class="group-info">
          <h3>评价: {{ record.conversation_id?.substring(0, 15) || 'N/A' }}...</h3>
          <span class="group-time">{{ formatDate(record.CreatedAt) }}</span>
        </div>
        <button
          @click="viewDetails(record)"
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
      </div>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  records: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['view-details'])

// 查看详情
const viewDetails = (record) => {
  // 根据 conversation_id 查找同一会话的所有记录
  const relatedRecords = props.records.filter(
    r => r.conversation_id === record.conversation_id
  )
  
  const sortedRecords = relatedRecords.sort((a, b) => {
    // student 记录在前，assistant 记录在后
    if (a.role === 'student' && b.role === 'assistant') return -1
    if (a.role === 'assistant' && b.role === 'student') return 1
    return 0
  })
  
  // 提取首次提交的题目描述和代码
  const initialSubmission = extractInitialSubmission(relatedRecords)
  
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
/* 公共样式已在 common.css 中定义 */
</style>
