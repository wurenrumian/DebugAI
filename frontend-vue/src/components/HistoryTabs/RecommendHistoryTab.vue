<template>
  <div class="records-list">
    <div
      v-for="record in records.filter(r => r.role === 'assistant')"
      :key="record.ID"
      class="record-group card"
    >
      <div class="group-header">
        <div class="group-info">
          <h3>推荐记录</h3>
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
          <span class="stat-icon">📚</span>
          题目推荐
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
  emit('view-details', {
    records: [record],
    initialSubmission: null,
    type: 'recommend'
  })
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
