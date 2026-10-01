<template>
  <ul class="record-list" aria-label="题目推荐列表">
    <RecordListItem
      v-for="record in records.filter(r => r.role === 'assistant')"
      :key="record.ID"
      icon="target"
      title="题目推荐"
      :time="formatDate(record.CreatedAt)"
      :timestamp="record.CreatedAt"
      @view="viewDetails(record)"
    />
  </ul>
</template>

<script setup>
import RecordListItem from './RecordListItem.vue'
import { formatRelativeTime } from '../../utils/format'

defineProps({
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
const formatDate = formatRelativeTime
</script>
