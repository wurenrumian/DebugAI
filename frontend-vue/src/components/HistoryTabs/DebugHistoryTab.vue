<template>
  <div class="records-list">
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

// 查看详情
const viewDetails = (group) => {
  // 按轮次排序，每轮内学生在前，AI在后
  const sortedRecords = group.records.sort((a, b) => {
    const aRound = getField(a, 'round_number', 'RoundNumber')
    const bRound = getField(b, 'round_number', 'RoundNumber')
    if (aRound !== bRound) {
      return aRound - bRound
    }
    // 同一轮内：学生(role=student)在前，AI(role=assistant)在后
    const aRole = getField(a, 'role', 'Role')
    const bRole = getField(b, 'role', 'Role')
    return aRole === 'student' ? -1 : 1
  })
  
  // 提取首次提交的题目描述和代码
  const initialSubmission = extractInitialSubmission(group.records)
  
  emit('view-details', {
    records: sortedRecords,
    initialSubmission,
    type: 'debug'
  })
}

// 从记录中提取首次提交的题目描述和代码
const extractInitialSubmission = (records) => {
  // 优先找 round_number 为 1 的学生记录（debug类型）
  let firstRecord = records.find(r => getField(r, 'round_number', 'RoundNumber') === 1 && getField(r, 'role', 'Role') === 'student')
  
  // 如果没找到，尝试找第一个学生记录（evaluate类型）
  if (!firstRecord) {
    firstRecord = records.find(r => getField(r, 'role', 'Role') === 'student')
  }
  
  const requestPayload = getField(firstRecord, 'request_payload', 'RequestPayload')
  if (firstRecord && requestPayload) {
    try {
      const req = typeof requestPayload === 'string'
        ? JSON.parse(requestPayload)
        : requestPayload
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

// 兼容大小写字段名的辅助函数
const getField = (record, lowercaseField, uppercaseField) => {
  if (!record) return null
  return record[lowercaseField] !== undefined ? record[lowercaseField] : record[uppercaseField]
}

// 按会话分组的记录（调试）
const groupedRecords = computed(() => {
  // 过滤出调试记录：round_number > 0 且 conversation_id 以 conv_ 或 dbg_ 开头
  const debugRecords = props.records.filter(r => {
    const roundNum = getField(r, 'round_number', 'RoundNumber')
    const convId = getField(r, 'conversation_id', 'ConversationID') || ''
    return roundNum > 0 && (convId.startsWith('conv_') || convId.startsWith('dbg_'))
  })
  
  const groups = {}
  
  debugRecords.forEach(record => {
    const convId = getField(record, 'conversation_id', 'ConversationID')
    const createdAt = getField(record, 'created_at', 'CreatedAt')
    const roundNum = getField(record, 'round_number', 'RoundNumber')
    
    if (!groups[convId]) {
      groups[convId] = {
        conversation_id: convId,
        records: [],
        latest_time: new Date(createdAt).getTime(),
        max_round: 0
      }
    }
    groups[convId].records.push(record)
    groups[convId].max_round = Math.max(groups[convId].max_round, roundNum)
    groups[convId].latest_time = Math.max(
      groups[convId].latest_time,
      new Date(createdAt).getTime()
    )
  })
  
  return Object.values(groups).sort((a, b) => b.latest_time - a.latest_time)
})

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