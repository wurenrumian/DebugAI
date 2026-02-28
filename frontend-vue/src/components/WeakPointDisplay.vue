<template>
  <div class="weak-point-display">
    <!-- 图表区域 -->
    <div v-if="showCharts && chartPosition === 'top'" class="charts-section">
      <div class="chart-item">
        <v-chart class="chart" :option="pieOption" autoresize />
      </div>
      <div class="chart-item">
        <v-chart class="chart" :option="barOption" autoresize />
      </div>
    </div>

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
    
    <!-- 图表区域（底部） -->
    <div v-if="showCharts && chartPosition === 'bottom'" class="charts-section">
      <div class="chart-item">
        <v-chart class="chart" :option="pieOption" autoresize />
      </div>
      <div class="chart-item">
        <v-chart class="chart" :option="barOption" autoresize />
      </div>
    </div>

    <!-- 导出按钮 -->
    <div v-if="showCharts" class="export-section">
      <button class="export-btn" @click="exportToCSV">导出 CSV</button>
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
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
} from 'echarts/components'

// 注册 ECharts 组件
use([
  CanvasRenderer,
  PieChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
])

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
  },
  // 新增：是否显示图表
  showCharts: {
    type: Boolean,
    default: false
  },
  // 新增：图表位置
  chartPosition: {
    type: String,
    default: 'top'
  },
  // 新增：柱状图显示前 N 个
  topN: {
    type: Number,
    default: 15
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

// ========== 图表相关逻辑 ==========

// 饼图数据
const pieChartData = computed(() => {
  const groups = groupedWeakPoints.value
  const categories = Object.keys(groups)
  return {
    categories,
    values: categories.map(cat => groups[cat].length)
  }
})

// 柱状图数据
const barChartData = computed(() => {
  const sorted = [...props.weakPoints].sort((a, b) => b.count - a.count)
  const topN = sorted.slice(0, props.topN)
  return {
    keywords: topN.map(wp => wp.keyword),
    counts: topN.map(wp => wp.count),
    categories: topN.map(wp => wp.category)
  }
})

// 饼图配置
const pieOption = computed(() => {
  const data = pieChartData.value
  if (data.categories.length === 0) {
    return {
      title: { text: '薄弱点分类分布', left: 'center', textStyle: { fontSize: 14, color: '#333' } },
      series: [{ type: 'pie', data: [] }]
    }
  }
  return {
    title: {
      text: '薄弱点分类分布',
      left: 'center',
      textStyle: { fontSize: 14, color: '#333' }
    },
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} 个关键词 ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left',
      top: 'middle'
    },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      data: data.categories.map((cat, idx) => ({
        value: data.values[idx],
        name: cat
      })),
      emphasis: {
        itemStyle: {
          shadowBlur: 10,
          shadowOffsetX: 0,
          shadowColor: 'rgba(0, 0, 0, 0.5)'
        }
      }
    }]
  }
})

// 柱状图配置
const barOption = computed(() => {
  const data = barChartData.value
  if (data.keywords.length === 0) {
    return {
      title: { text: `Top ${props.topN} 薄弱点关键词`, left: 'center', textStyle: { fontSize: 14, color: '#333' } },
      series: [{ type: 'bar', data: [] }]
    }
  }
  return {
    title: {
      text: `Top ${props.topN} 薄弱点关键词`,
      left: 'center',
      textStyle: { fontSize: 14, color: '#333' }
    },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params) => {
        const item = params[0]
        return `${item.name}<br/>分类：${data.categories[item.dataIndex]}<br/>出现次数：${item.value}`
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: 60,
      containLabel: true
    },
    xAxis: {
      type: 'value',
      name: '出现次数'
    },
    yAxis: {
      type: 'category',
      data: data.keywords.slice().reverse(),
      name: '关键词'
    },
    series: [{
      type: 'bar',
      data: data.counts.slice().reverse(),
      itemStyle: {
        color: '#409eff'
      },
      label: {
        show: true,
        position: 'right'
      }
    }]
  }
})

// CSV 导出功能
const exportToCSV = () => {
  const headers = ['keyword', 'category', 'count', 'description']
  const rows = props.weakPoints.map(wp => [
    wp.keyword,
    wp.category,
    wp.count,
    wp.description || ''
  ])

  // 添加 BOM 以支持 Excel 中文
  const BOM = '\uFEFF'
  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(','))
  ].join('\n')

  const blob = new Blob([BOM + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `weak_points_${new Date().toISOString().split('T')[0]}.csv`
  link.click()
  URL.revokeObjectURL(link.href)
}
</script>

<style scoped>
.weak-point-display {
  width: 100%;
}

/* 图表区域样式 */
.charts-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
  margin: 16px 0;
  padding: 16px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #ebeef5;
}

.chart-item {
  height: 320px;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  padding: 12px;
}

.chart {
  height: 100%;
  width: 100%;
}

.export-section {
  margin-top: 16px;
  text-align: right;
}

.export-btn {
  padding: 8px 16px;
  background-color: #409eff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.export-btn:hover {
  background-color: #66b1ff;
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
