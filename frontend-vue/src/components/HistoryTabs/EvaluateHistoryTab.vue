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
          @click="$emit('view-details', record)"
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
defineProps({
  records: {
    type: Array,
    default: () => []
  }
})

defineEmits(['view-details'])

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