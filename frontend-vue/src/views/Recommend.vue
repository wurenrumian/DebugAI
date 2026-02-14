<template>
  <div class="ai-debug-container">
    <div class="ai-debug-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📚 AI 题目推荐</h1>
    </div>
    
    <div class="ai-debug-content">
      <!-- 左侧：薄弱点选择 -->
      <div class="left-panel">
        <div class="card">
          <h2 class="subtitle">📊 您的薄弱点</h2>
          <p class="description">基于您的历史调试记录自动统计的薄弱知识点</p>
          
          <div v-if="loadingWeakPoints" class="loading-small">
            加载中...
          </div>
          
          <div v-else-if="userWeakPoints.length === 0" class="empty-state">
            <p>暂无薄弱点记录</p>
            <p class="hint">完成更多 AI 调试后，系统会自动统计您的薄弱点</p>
          </div>
          
          <div v-else class="weak-points-list">
            <div 
              v-for="(wp, index) in userWeakPoints" 
              :key="index"
              :class="['weak-point-item', { selected: selectedWeakPoints.includes(wp.keyword) }]"
              @click="toggleWeakPoint(wp.keyword)"
            >
              <span class="keyword">{{ wp.keyword }}</span>
              <span class="count">出现 {{ wp.count }} 次</span>
            </div>
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
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { aiAPI } from '@/api'

const authStore = useAuthStore()

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
  try {
    const response = await aiAPI.getWeakPoints()
    if (response.data && response.data.length > 0) {
      // 转换为关键词并排序
      const weakPointMap = new Map()
      for (const wp of response.data) {
        const existing = weakPointMap.get(wp.weak_point_id)
        if (existing) {
          existing.count += wp.count
        } else {
          // 这里需要通过 ID 获取关键词
          // 由于 API 返回的是关联表数据，我们使用一个简化的方法
          weakPointMap.set(wp.weak_point_id, {
            keyword: `知识点${wp.weak_point_id}`,
            count: wp.count
          })
        }
      }
      
      // 尝试获取前5个薄弱点
      try {
        const topResponse = await aiAPI.getTopWeakPoints()
        if (topResponse.data && topResponse.data.length > 0) {
          selectedWeakPoints.value = topResponse.data
          userWeakPoints.value = topResponse.data.map(kw => ({
            keyword: kw,
            count: 1
          }))
        }
      } catch (e) {
        console.log('No top weak points yet')
      }
    }
  } catch (error) {
    console.error('Failed to fetch weak points:', error)
  } finally {
    loadingWeakPoints.value = false
  }
}

// 切换薄弱点选择
const toggleWeakPoint = (keyword) => {
  const index = selectedWeakPoints.value.indexOf(keyword)
  if (index === -1) {
    selectedWeakPoints.value.push(keyword)
  } else {
    selectedWeakPoints.value.splice(index, 1)
  }
}

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
.ai-debug-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.ai-debug-header {
  max-width: 1400px;
  margin: 0 auto 20px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.ai-debug-header h1 {
  margin: 10px 0 0;
  color: #333;
}

.back-link {
  color: #667eea;
  text-decoration: none;
  font-weight: 500;
}

.back-link:hover {
  text-decoration: underline;
}

.ai-debug-content {
  max-width: 1400px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.left-panel, .right-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
}

.subtitle {
  margin: 0 0 8px;
  font-size: 16px;
  color: #333;
  font-weight: 600;
}

.description {
  margin: 0 0 16px;
  color: #666;
  font-size: 14px;
}

.empty-state {
  text-align: center;
  padding: 20px;
  color: #999;
}

.empty-state .hint {
  font-size: 13px;
  margin-top: 8px;
}

.weak-points-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.weak-point-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f5f5f5;
  border-radius: 20px;
  cursor: pointer;
  transition: all 0.3s;
  border: 2px solid transparent;
}

.weak-point-item:hover {
  background: #e8e8e8;
}

.weak-point-item.selected {
  background: #e6f7ff;
  border-color: #1890ff;
}

.weak-point-item .keyword {
  font-weight: 500;
  color: #333;
}

.weak-point-item .count {
  font-size: 12px;
  color: #999;
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

.btn {
  padding: 14px 28px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: #e0e0e0;
  color: #333;
}

.btn-secondary:hover {
  background: #d0d0d0;
}

.start-btn, .reset-btn {
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

.empty-dialogue {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  color: #999;
}

.empty-dialogue .hint {
  font-size: 14px;
  margin-top: 8px;
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

.loading-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 12px;
  margin-top: 16px;
}

.loading-item .dialogue-avatar {
  font-size: 32px;
}

.loading-item .dialogue-bubble {
  flex: 1;
}

.loading-item .dialogue-label {
  font-size: 14px;
  color: #667eea;
  font-weight: 600;
  margin-bottom: 8px;
}

.loading-item .dialogue-text {
  color: #666;
}

.loading .dots {
  animation: blink 1.5s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.loading-small {
  text-align: center;
  padding: 20px;
  color: #999;
}

.error-message {
  background: #fee;
  border: 1px solid #fcc;
  color: #c33;
  padding: 12px;
  border-radius: 8px;
  margin-top: 16px;
}

@media (max-width: 900px) {
  .ai-debug-content {
    grid-template-columns: 1fr;
  }
}
</style>
