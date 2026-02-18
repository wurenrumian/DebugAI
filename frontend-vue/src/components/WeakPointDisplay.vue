<template>
  <div class="weak-point-display">
    <!-- 按 category 分组展示 -->
    <div v-for="(points, category) in groupedWeakPoints" :key="category" class="category-group">
      <h4 class="category-title">{{ category }}</h4>
      <div class="weak-points-container">
        <div 
          v-for="(wp, index) in getDisplayPoints(points)" 
          :key="index"
          :class="['weak-point-item', { 
            selected: isSelected(wp.keyword),
            selectable: selectable
          }]"
          @click="selectable && toggleSelect(wp.keyword)"
        >
          <span class="keyword">{{ wp.keyword }}</span>
          <span class="count">({{ wp.count }}次)</span>
          
          <!-- 描述 tooltip -->
          <span 
            v-if="showDescription && wp.description" 
            class="description-tooltip"
            :title="wp.description"
          >?</span>
        </div>
        
        <!-- 查看更多 -->
        <span 
          v-if="maxDisplay > 0 && points.length > maxDisplay" 
          class="view-more"
          @click="showAll[category] = !showAll[category]"
        >
          {{ showAll[category] ? '收起' : `查看更多 (${points.length - maxDisplay})` }}
        </span>
      </div>
    </div>
    
    <!-- 空状态 -->
    <div v-if="Object.keys(groupedWeakPoints).length === 0" class="empty-state">
      <slot name="empty">
        <p>暂无薄弱点数据</p>
      </slot>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  // 薄弱点数据: [{keyword, category, count, description?}]
  weakPoints: {
    type: Array,
    default: () => []
  },
  // 是否可选择
  selectable: {
    type: Boolean,
    default: false
  },
  // 选中的关键词 (v-model)
  selected: {
    type: Array,
    default: () => []
  },
  // 是否显示描述
  showDescription: {
    type: Boolean,
    default: false
  },
  // 限制显示数量，0 表示全部显示
  maxDisplay: {
    type: Number,
    default: 0
  }
})

const emit = defineEmits(['update:selected'])

// 展开/收起状态
const showAll = ref({})

// 按 category 分组
const groupedWeakPoints = computed(() => {
  const groups = {}
  for (const wp of props.weakPoints) {
    const category = wp.category || '未分类'
    if (!groups[category]) {
      groups[category] = []
    }
    groups[category].push(wp)
  }
  // 每个分组内按 count 降序排序
  for (const key in groups) {
    groups[key].sort((a, b) => b.count - a.count)
  }
  return groups
})

// 获取实际显示的 points
const getDisplayPoints = (points) => {
  if (props.maxDisplay > 0 && !showAll.value[points[0]?.category]) {
    return points.slice(0, props.maxDisplay)
  }
  return points
}

// 检查是否选中
const isSelected = (keyword) => {
  return props.selected.includes(keyword)
}

// 切换选择
const toggleSelect = (keyword) => {
  const newSelected = [...props.selected]
  const index = newSelected.indexOf(keyword)
  if (index === -1) {
    newSelected.push(keyword)
  } else {
    newSelected.splice(index, 1)
  }
  emit('update:selected', newSelected)
}

// 监听 maxDisplay 变化，初始化 showAll
watch(() => props.maxDisplay, () => {
  showAll.value = {}
})
</script>

<style scoped>
.weak-point-display {
  width: 100%;
}

.category-group {
  margin-bottom: 16px;
}

.category-title {
  font-size: 14px;
  font-weight: 600;
  color: #666;
  margin-bottom: 8px;
  padding-left: 4px;
  border-left: 3px solid #409eff;
}

.weak-points-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.weak-point-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background-color: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  font-size: 14px;
  cursor: default;
  transition: all 0.2s ease;
}

.weak-point-item.selectable {
  cursor: pointer;
}

.weak-point-item.selectable:hover {
  border-color: #409eff;
  background-color: #ecf5ff;
}

.weak-point-item.selected {
  background-color: #409eff;
  border-color: #409eff;
  color: white;
}

.weak-point-item.selected .count {
  color: rgba(255, 255, 255, 0.8);
}

.keyword {
  font-weight: 500;
}

.count {
  font-size: 12px;
  color: #909399;
}

.description-tooltip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  margin-left: 4px;
  background-color: #909399;
  color: white;
  border-radius: 50%;
  font-size: 10px;
  cursor: help;
}

.view-more {
  color: #409eff;
  font-size: 13px;
  cursor: pointer;
  padding: 4px 8px;
}

.view-more:hover {
  text-decoration: underline;
}

.empty-state {
  text-align: center;
  padding: 24px;
  color: #909399;
}
</style>
