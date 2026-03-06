<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📚 AI 题目推荐</h1>
    </div>
    
    <div class="content-wrapper ai-debug-content">
      <!-- 左侧：薄弱点选择 -->
      <div class="left-panel">
        <div class="card">
          <h2 class="subtitle">📊 您的薄弱点</h2>
          <p class="description">基于您的历史调试记录自动统计的薄弱知识点</p>
          
          <!-- 筛选控件 -->
          <div class="filter-controls">
            <div class="filter-row">
              <div class="filter-item">
                <label>开始日期：</label>
                <input type="date" v-model="startDate" />
              </div>
              <div class="filter-item">
                <label>结束日期：</label>
                <input type="date" v-model="endDate" />
              </div>
            </div>
            <div class="filter-row">
              <div class="filter-item">
                <label>显示前：</label>
                <input type="number" v-model.number="topK" min="0" max="20" style="width: 50px" />
                <span class="filter-hint">（0 = 全部）</span>
              </div>
            </div>
          </div>
          
          <div v-if="loadingWeakPoints" class="loading-small">
            加载中...
          </div>
          
          <div v-else-if="userWeakPoints.length === 0" class="empty-state">
            <p>暂无薄弱点记录</p>
            <p class="hint">完成更多 AI 调试后，系统会自动统计您的薄弱点</p>
          </div>
          
          <div v-else class="weak-points-container">
            <WeakPointDisplay
              :weakPoints="userWeakPoints"
              :selectable="true"
              v-model:selected="selectedWeakPoints"
              :showDescription="true"
              :maxDisplay="5"
              :showCharts="true"
              chartPosition="top"
              :topN="10"
            />
          </div>
        </div>
        
        <div class="card">
          <h2 class="subtitle">🎯 推荐设置</h2>
          <div class="setting-item">
            <label>推荐数量：</label>
            <select v-model="maxRecommendations">
              <option :value="3">3 个</option>
              <option :value="5">5 个</option>
              <option :value="8">8 个</option>
              <option :value="10">10 个</option>
            </select>
          </div>
        </div>
        
        <button 
          @click="submitRecommend" 
          class="btn btn-primary start-btn" 
          :disabled="loading || !canSubmit"
        >
          {{ loading ? '推荐中...' : '获取推荐' }}
        </button>
        
        <button 
          v-if="hasResult" 
          @click="resetForm" 
          class="btn btn-secondary reset-btn"
        >
          重新推荐
        </button>
      </div>
      
      <!-- 右侧：推荐结果 -->
      <div class="right-panel">
        <div class="card dialogue-card">
          <div class="dialogue-header">
            <h2 class="subtitle">推荐结果</h2>
          </div>
          
          <div v-if="!hasResult" class="empty-dialogue">
            <p>暂无推荐结果</p>
            <p class="hint">请在左侧选择薄弱点，点击"获取推荐"</p>
          </div>
          
          <div v-else class="recommendation-result">
            <!-- 分析 -->
            <div class="analysis-section">
              <h3>📝 分析总结</h3>
              <p>{{ result.analysis }}</p>
            </div>
            
            <!-- 推荐列表 -->
            <div class="recommendations-list">
              <h3>🎯 推荐题目标签</h3>
              <div 
                v-for="(rec, index) in result.recommendations" 
                :key="index"
                class="recommendation-item"
              >
                <div class="rec-header">
                  <span class="rec-tag">{{ rec.tag }}</span>
                  <span class="rec-relevance" :class="getRelevanceClass(rec.relevance)">
                    相关度: {{ (rec.relevance * 100).toFixed(0) }}%
                  </span>
                </div>
                <div class="rec-reason">{{ rec.reason }}</div>
              </div>
            </div>
          </div>
          
          <div v-if="loading" class="loading-item">
            <div class="dialogue-avatar">🤖</div>
            <div class="dialogue-bubble">
              <div class="dialogue-label">AI 助手</div>
              <div class="dialogue-text loading">
                <span>正在思考</span>
                <span class="dots">...</span>
              </div>
            </div>
          </div>
        </div>
        
        <div v-if="errorMessage" class="error-message">
          {{ errorMessage }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useKeyboardShortcut } from '@/composables/useKeyboardShortcut'
import { useAuthStore } from '@/stores/auth'
import { aiAPI } from '@/api'
import WeakPointDisplay from '@/components/WeakPointDisplay.vue'

const authStore = useAuthStore()

// 获取今日日期字符串
const getTodayDate = () => {
  const today = new Date()
  const year = today.getFullYear()
  const month = String(today.getMonth() + 1).padStart(2, '0')
  const day = String(today.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 筛选条件 - 初始为今日
const startDate = ref(getTodayDate())
const endDate = ref(getTodayDate())
const topK = ref(0)  // 0 表示返回全部薄弱点

// 用户薄弱点
const userWeakPoints = ref([])

// 选中的薄弱点
const selectedWeakPoints = ref([])

// 推荐数量
const maxRecommendations = ref(5)

// 加载状态
const loading = ref(false)
const loadingWeakPoints = ref(false)

// 结果数据
const result = ref(null)

// 错误信息
const errorMessage = ref('')

// 是否有结果
const hasResult = computed(() => {
  return result.value !== null && result.value.recommendations?.length > 0
})

// 是否可以提交
const canSubmit = computed(() => {
  return selectedWeakPoints.value.length > 0
})

// 获取用户薄弱点
const fetchUserWeakPoints = async () => {
  loadingWeakPoints.value = true
  
  // 构建查询参数
  const params = {}
  if (startDate.value) params.start_date = startDate.value
  if (endDate.value) params.end_date = endDate.value
  
  try {
    if (topK.value > 0) {
      // 使用 Top K 接口
      const response = await aiAPI.getTopWeakPoints(params)
      if (response.data && response.data.length > 0) {
        userWeakPoints.value = response.data.slice(0, topK.value)
      } else {
        userWeakPoints.value = []
      }
    } else {
      // 使用全部薄弱点接口
      const response = await aiAPI.getWeakPoints(params)
      if (response.data && response.data.length > 0) {
        userWeakPoints.value = response.data
      } else {
        userWeakPoints.value = []
      }
    }
    // 清空选中状态
    selectedWeakPoints.value = []
  } catch (error) {
    console.error('Failed to fetch weak points:', error)
    userWeakPoints.value = []
  } finally {
    loadingWeakPoints.value = false
  }
}

// 监听筛选条件变化，自动重新获取数据
watch([startDate, endDate, topK], () => {
  fetchUserWeakPoints()
})

// 提交推荐
const submitRecommend = async () => {
  if (!canSubmit.value || loading.value) return
  
  loading.value = true
  errorMessage.value = ''
  
  // 构建 weak_points 字典
  const weakPoints = {}
  for (const wp of selectedWeakPoints.value) {
    weakPoints[wp] = 1
  }
  
  const requestData = {
    student_id: authStore.user.student_id,
    weak_points: weakPoints,
    max_recommendations: maxRecommendations.value
  }
  
  try {
    const response = await aiAPI.recommend(requestData)
    result.value = response
  } catch (error) {
    errorMessage.value = error.error || '推荐请求失败，请稍后重试'
    console.error('Recommend error:', error)
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  result.value = null
  errorMessage.value = ''
}

// Enter 键提交（表单验证通过时）
useKeyboardShortcut(['enter'], (event) => {
  // 如果在输入框内，不处理
  if (event.target.tagName === 'TEXTAREA' || event.target.tagName === 'INPUT') {
    return
  }
  if (!loading.value && canSubmit.value) {
    submitRecommend()
  }
})

// Ctrl+Enter 键提交
useKeyboardShortcut(['ctrl+enter'], () => {
  if (!loading.value && canSubmit.value) {
    submitRecommend()
  }
})

// 获取相关度样式类
const getRelevanceClass = (relevance) => {
  if (relevance >= 0.8) return 'relevance-high'
  if (relevance >= 0.5) return 'relevance-medium'
  return 'relevance-low'
}

// 页面加载时获取薄弱点
onMounted(() => {
  fetchUserWeakPoints()
})
</script>

<style scoped>
.ai-debug-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.left-panel,
.right-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.description {
  margin: 0 0 16px;
  color: #666;
  font-size: 14px;
}

.filter-controls {
  margin-bottom: 16px;
  padding: 12px;
  background-color: #f5f7fa;
  border-radius: 8px;
}

.filter-row {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.filter-row:last-child {
  margin-bottom: 0;
}

.filter-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-item label {
  color: #666;
  font-size: 14px;
  white-space: nowrap;
}

.filter-item input[type="date"],
.filter-item input[type="number"] {
  padding: 6px 10px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
}

.filter-item input[type="date"]:focus,
.filter-item input[type="number"]:focus {
  outline: none;
  border-color: #409eff;
}

.filter-item input[type="number"] {
  width: 60px;
}

.weak-points-container {
  min-height: 100px;
}

.setting-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.setting-item label {
  color: #666;
}

.setting-item select {
  padding: 8px 12px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
}

.setting-item select:focus {
  outline: none;
  border-color: #667eea;
}

.start-btn,
.reset-btn {
  width: 100%;
}

.dialogue-card {
  flex: 1;
  min-height: 500px;
  display: flex;
  flex-direction: column;
}

.dialogue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.recommendation-result {
  flex: 1;
  overflow-y: auto;
}

.analysis-section {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid #667eea;
  margin-bottom: 20px;
}

.analysis-section h3 {
  margin: 0 0 8px;
  color: #333;
}

.analysis-section p {
  margin: 0;
  color: #666;
  line-height: 1.6;
}

.recommendations-list h3 {
  margin: 0 0 12px;
  color: #333;
}

.recommendation-item {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 8px;
  margin-bottom: 12px;
}

.rec-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.rec-tag {
  font-weight: 600;
  color: #333;
  font-size: 16px;
}

.rec-relevance {
  font-size: 13px;
  padding: 4px 8px;
  border-radius: 4px;
}

.relevance-high {
  background: #d4edda;
  color: #155724;
}

.relevance-medium {
  background: #fff3cd;
  color: #856404;
}

.relevance-low {
  background: #f8d7da;
  color: #721c24;
}

.rec-reason {
  color: #666;
  font-size: 14px;
  line-height: 1.5;
}

@media (max-width: 900px) {
  .ai-debug-content {
    grid-template-columns: 1fr;
  }
}
</style>
