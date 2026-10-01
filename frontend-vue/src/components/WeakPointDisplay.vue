<template>
  <div class="weak-point-display">
    <!-- 图表区域（顶部） -->
    <div v-if="showCharts && chartPosition === 'top' && hasData" class="charts-section">
      <figure class="chart-item">
        <figcaption class="chart-title">薄弱点分类分布</figcaption>
        <v-chart class="chart" :option="pieOption" autoresize />
      </figure>
      <figure class="chart-item">
        <figcaption class="chart-title">Top {{ topN }} 薄弱点关键词</figcaption>
        <v-chart class="chart" :option="barOption" autoresize />
      </figure>
    </div>

    <!-- 按 category 分组展示 -->
    <div v-if="hasData" class="category-list">
      <section v-for="(points, category) in groupedWeakPoints" :key="category" class="category-group">
        <h4 class="category-title">
          {{ category }}
          <span class="category-count">{{ points.length }}</span>
        </h4>
        <div class="weak-points-container" :role="selectable ? 'group' : undefined" :aria-label="selectable ? `${category}：选择薄弱点` : undefined">
          <component
            :is="selectable ? 'button' : 'span'"
            v-for="(wp, index) in getDisplayPoints(points)"
            :key="index"
            :type="selectable ? 'button' : undefined"
            :aria-pressed="selectable ? (isSelected(wp.keyword) ? 'true' : 'false') : undefined"
            :class="['weak-point-item', {
              selected: isSelected(wp.keyword),
              selectable: selectable
            }]"
            :title="showDescription && wp.description ? wp.description : undefined"
            @click="selectable && toggleSelect(wp.keyword)"
          >
            <AppIcon v-if="selectable && isSelected(wp.keyword)" name="check" :size="13" :stroke-width="2.6" />
            <span class="keyword">{{ wp.keyword }}</span>
            <span class="count">{{ wp.count }}次</span>
            <span
              v-if="showDescription && wp.description"
              class="description-tooltip"
              aria-hidden="true"
            >?</span>
            <span v-if="showDescription && wp.description" class="sr-only">：{{ wp.description }}</span>
          </component>

          <!-- 查看更多 -->
          <button
            v-if="maxDisplay > 0 && points.length > maxDisplay"
            type="button"
            class="view-more"
            :aria-expanded="showAll[category] ? 'true' : 'false'"
            @click="showAll[category] = !showAll[category]"
          >
            {{ showAll[category] ? '收起' : `查看更多 (${points.length - maxDisplay})` }}
          </button>
        </div>
      </section>
    </div>

    <!-- 图表区域（底部） -->
    <div v-if="showCharts && chartPosition === 'bottom' && hasData" class="charts-section">
      <figure class="chart-item">
        <figcaption class="chart-title">薄弱点分类分布</figcaption>
        <v-chart class="chart" :option="pieOption" autoresize />
      </figure>
      <figure class="chart-item">
        <figcaption class="chart-title">Top {{ topN }} 薄弱点关键词</figcaption>
        <v-chart class="chart" :option="barOption" autoresize />
      </figure>
    </div>

    <!-- 导出按钮 -->
    <div v-if="showCharts && hasData" class="export-section">
      <AppButton size="sm" icon="download" @click="exportToCSV">导出 CSV</AppButton>
    </div>

    <!-- 空状态 -->
    <div v-if="!hasData" class="empty-state">
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
  TooltipComponent,
  LegendComponent,
  GridComponent
} from 'echarts/components'
import { escapeHtml } from '../utils/html'
import { themeState, readCssVar } from '../utils/theme'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'

// 注册 ECharts 组件
use([
  CanvasRenderer,
  PieChart,
  BarChart,
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
  // 是否显示图表
  showCharts: {
    type: Boolean,
    default: false
  },
  // 图表位置
  chartPosition: {
    type: String,
    default: 'top'
  },
  // 柱状图显示前 N 个
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

const hasData = computed(() => Object.keys(groupedWeakPoints.value).length > 0)

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

// 分类色最多 8 个，按固定顺序使用；超出的分类合并为“其他”（不循环复用颜色）
const MAX_CATEGORIES = 8

// 当前主题下的图表配色（依赖 themeState.resolved，切换主题时重新计算并重绘）
const chartTheme = computed(() => {
  // 读取 resolved 以建立响应式依赖；CSS 变量已随 data-theme 同步更新
  const resolved = themeState.resolved
  return {
    resolved,
    palette: Array.from({ length: MAX_CATEGORIES }, (_, i) => readCssVar(`--chart-${i + 1}`, '#2a78d6')),
    text: readCssVar('--color-text', '#1d2433'),
    label: readCssVar('--chart-label', '#485263'),
    axis: readCssVar('--chart-axis', '#667085'),
    grid: readCssVar('--chart-grid', '#e8ebf0'),
    surface: readCssVar('--color-surface', '#ffffff'),
    border: readCssVar('--color-border', '#e3e6eb'),
    font: readCssVar('--font-sans', 'sans-serif')
  }
})

const tooltipStyle = (theme) => ({
  backgroundColor: theme.surface,
  borderColor: theme.border,
  borderWidth: 1,
  padding: [8, 12],
  textStyle: { color: theme.text, fontSize: 13, fontFamily: theme.font },
  extraCssText: 'box-shadow: 0 6px 20px rgba(0,0,0,0.12); border-radius: 8px;'
})

// 饼图数据
const pieChartData = computed(() => {
  const groups = groupedWeakPoints.value
  const entries = Object.keys(groups)
    .map(cat => ({ name: cat, value: groups[cat].length }))
    .sort((a, b) => b.value - a.value)
  if (entries.length <= MAX_CATEGORIES) return entries
  const head = entries.slice(0, MAX_CATEGORIES - 1)
  const rest = entries.slice(MAX_CATEGORIES - 1).reduce((sum, e) => sum + e.value, 0)
  return [...head, { name: '其他', value: rest }]
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
  const theme = chartTheme.value
  const data = pieChartData.value
  return {
    color: theme.palette,
    textStyle: { fontFamily: theme.font },
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} 个关键词 ({d}%)',
      ...tooltipStyle(theme)
    },
    // 最多 8 项，平铺自动换行（不分页）
    legend: {
      type: 'plain',
      bottom: 0,
      left: 4,
      right: 4,
      icon: 'circle',
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 12,
      textStyle: { color: theme.label, fontSize: 12 }
    },
    series: [{
      type: 'pie',
      radius: ['40%', '62%'],
      center: ['50%', '40%'],
      avoidLabelOverlap: true,
      label: { show: false },
      labelLine: { show: false },
      itemStyle: {
        borderColor: theme.surface,
        borderWidth: 2,
        borderRadius: 4
      },
      emphasis: {
        scale: true,
        scaleSize: 4,
        label: {
          show: true,
          formatter: '{b}\n{c}',
          fontSize: 13,
          fontWeight: 600,
          color: theme.text,
          lineHeight: 18
        }
      },
      data
    }]
  }
})

// 柱状图配置
const barOption = computed(() => {
  const theme = chartTheme.value
  const data = barChartData.value
  return {
    textStyle: { fontFamily: theme.font },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow', shadowStyle: { color: theme.grid, opacity: 0.6 } },
      ...tooltipStyle(theme),
      formatter: (params) => {
        const item = params[0]
        // 名称 / 分类来自 AI 输出，拼接 HTML 前必须转义
        return `${escapeHtml(item.name)}<br/>分类：${escapeHtml(data.categories[item.dataIndex])}<br/>出现次数：${escapeHtml(item.value)}`
      }
    },
    grid: {
      left: 8,
      right: 36,
      bottom: 8,
      top: 8,
      containLabel: true
    },
    xAxis: {
      type: 'value',
      minInterval: 1,
      axisLabel: { color: theme.axis, fontSize: 11 },
      splitLine: { lineStyle: { color: theme.grid } }
    },
    yAxis: {
      type: 'category',
      data: data.keywords.slice().reverse(),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: theme.grid } },
      axisLabel: {
        color: theme.label,
        fontSize: 12,
        width: 110,
        overflow: 'truncate'
      }
    },
    series: [{
      type: 'bar',
      data: data.counts.slice().reverse(),
      barMaxWidth: 16,
      itemStyle: {
        color: theme.palette[0],
        borderRadius: [0, 4, 4, 0]
      },
      label: {
        show: true,
        position: 'right',
        color: theme.label,
        fontSize: 11
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
  const BOM = '﻿'
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
  min-width: 0;
}

/* 图表区域 */
.charts-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 300px), 1fr));
  gap: var(--space-4);
  margin-bottom: var(--space-5);
}

.chart-item {
  display: flex;
  flex-direction: column;
  min-width: 0;
  height: 320px;
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.chart-title {
  margin-bottom: var(--space-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
}

.chart {
  flex: 1;
  min-height: 0;
  width: 100%;
}

.export-section {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-4);
}

.category-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.category-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
}

.category-count {
  padding: 0 6px;
  font-size: 11px;
  font-weight: var(--fw-medium);
  line-height: 18px;
  color: var(--color-text-muted);
  background: var(--color-surface-3);
  border-radius: var(--radius-full);
}

.weak-points-container {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  align-items: center;
}

.weak-point-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-height: 32px;
  padding: 4px 12px;
  font-size: var(--fs-base);
  line-height: 1.4;
  color: var(--color-text);
  text-align: left;
  background: var(--color-surface);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-full);
  cursor: default;
  transition:
    background-color var(--duration-fast) var(--ease),
    border-color var(--duration-fast) var(--ease),
    color var(--duration-fast) var(--ease);
}

.weak-point-item.selectable {
  cursor: pointer;
}

.weak-point-item.selectable:hover {
  border-color: var(--color-primary);
  background: var(--color-primary-soft);
}

.weak-point-item.selected {
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
  box-shadow: inset 0 0 0 1px var(--color-primary);
}

.keyword {
  font-weight: var(--fw-medium);
  word-break: break-word;
}

.count {
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.description-tooltip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  font-size: 10px;
  font-weight: var(--fw-semibold);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border-strong);
  border-radius: 50%;
  cursor: help;
}

.view-more {
  padding: 4px 8px;
  font-size: var(--fs-sm);
  color: var(--color-primary-text);
  border-radius: var(--radius-sm);
}

.view-more:hover {
  text-decoration: underline;
  text-underline-offset: 2px;
}

.empty-state {
  padding: var(--space-6);
  text-align: center;
  color: var(--color-text-muted);
}

@media (max-width: 640px) {
  .chart-item {
    height: 280px;
    padding: var(--space-3);
  }
}
</style>
