<template>
  <ul class="record-list" aria-label="代码评价列表">
    <RecordListItem
      v-for="group in groupedRecords"
      :key="group.conversation_id"
      icon="evaluate"
      :title="`评价 ${group.conversation_id?.substring(0, 15) || 'N/A'}...`"
      :full-title="group.conversation_id || ''"
      :time="formatDate(group.latest_time)"
      :timestamp="group.latest_time"
      :stats="group.records.length > 1 ? ['代码评价', `${group.records.length} 条记录`] : ['代码评价']"
      @view="viewDetails(group)"
    />
  </ul>
</template>

<script setup>
import { computed } from 'vue'
import RecordListItem from './RecordListItem.vue'
import { formatRelativeTime } from '../../utils/format'

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
const formatDate = formatRelativeTime
</script>
