<template>
  <div class="ai-debug-container">
    <div class="ai-debug-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📝 AI 代码评价</h1>
    </div>
    
    <div class="ai-debug-content">
      <!-- 左侧：输入区域 -->
      <div class="left-panel">
        <div class="card">
          <h2 class="subtitle">题目描述</h2>
          <textarea 
            v-model="formData.problem_description" 
            placeholder="请输入题目描述..."
            rows="4"
          ></textarea>
        </div>
        
        <div class="card">
          <h2 class="subtitle">代码</h2>
          <textarea 
            v-model="formData.code" 
            placeholder="请输入需要评价的代码（C/C++）..."
            rows="12"
            class="code-input"
          ></textarea>
        </div>
        
        <div class="card">
          <h2 class="subtitle">测试点 (可选)</h2>
          <textarea 
            v-model="testPointsText" 
            placeholder="请输入测试点，每行一个，例如：&#10;输入: 1 2 3&#10;输出: 6&#10;状态: Accepted"
            rows="4"
          ></textarea>
        </div>
        
        <button 
          @click="submitEvaluate" 
          class="btn btn-primary start-btn" 
          :disabled="loading || !canSubmit"
        >
          {{ loading ? '评价中...' : '提交评价' }}
        </button>
        
        <button 
          v-if="hasResult" 
          @click="resetForm" 
          class="btn btn-secondary reset-btn"
        >
          重新评价
        </button>
      </div>
      
      <!-- 右侧：评价结果 -->
      <div class="right-panel">
        <div class="card dialogue-card">
          <div class="dialogue-header">
            <h2 class="subtitle">评价结果</h2>
          </div>
          
          <div v-if="!hasResult" class="empty-dialogue">
            <p>暂无评价结果</p>
            <p class="hint">请在左侧输入代码和题目描述，点击"提交评价"</p>
          </div>
          
          <div v-else class="evaluation-result">
            <!-- 整体评价 -->
            <div class="eval-section overall">
              <h3>📊 整体评价</h3>
              <p>{{ result.overall_evaluation }}</p>
            </div>
            
            <!-- 各项评分 -->
            <div class="eval-grid">
              <div class="eval-item">
                <div class="eval-label">✅ 功能正确</div>
                <div class="eval-score" :class="getScoreClass(result.functional_correctness?.grade)">
                  {{ result.functional_correctness?.grade || 'N/A' }}
                </div>
                <div class="eval-comment">{{ result.functional_correctness?.comment }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">🔍 逻辑严谨</div>
                <div class="eval-score" :class="getScoreClass(result.logical_rigor?.grade)">
                  {{ result.logical_rigor?.grade || 'N/A' }}
                </div>
                <div class="eval-comment">{{ result.logical_rigor?.comment }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">⚡ 算法效率</div>
                <div class="eval-score" :class="getScoreClass(result.algorithm_quality?.grade)">
                  {{ result.algorithm_quality?.grade || 'N/A' }}
                </div>
                <div class="eval-comment">{{ result.algorithm_quality?.comment }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">📐 结构规范</div>
                <div class="eval-score" :class="getScoreClass(result.structural_normativity?.grade)">
                  {{ result.structural_normativity?.grade || 'N/A' }}
                </div>
                <div class="eval-comment">{{ result.structural_normativity?.comment }}</div>
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
import { ref, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { aiAPI } from '@/api'

const authStore = useAuthStore()

// 表单数据
const formData = ref({
  problem_description: '',
  code: ''
})

// 测试点文本
const testPointsText = ref('')

// 加载状态
const loading = ref(false)

// 结果数据
const result = ref(null)

// 错误信息
const errorMessage = ref('')

// 是否可以提交
const canSubmit = computed(() => {
  return formData.value.code.trim() && formData.value.problem_description.trim()
})

// 是否有结果
const hasResult = computed(() => {
  return result.value !== null
})

// 解析测试点
const parseTestPoints = () => {
  if (!testPointsText.value.trim()) {
    return []
  }
  
  const lines = testPointsText.value.split('\n').filter(line => line.trim())
  const testPoints = []
  let currentInput = ''
  let currentOutput = ''
  let currentStatus = ''
  
  for (const line of lines) {
    if (line.startsWith('输入:')) {
      currentInput = line.substring(3).trim()
    } else if (line.startsWith('输出:')) {
      currentOutput = line.substring(3).trim()
    } else if (line.startsWith('状态:')) {
      currentStatus = line.substring(3).trim()
      testPoints.push({
        input: currentInput,
        output: currentOutput,
        status: currentStatus
      })
      currentInput = ''
      currentOutput = ''
      currentStatus = ''
    }
  }
  
  return testPoints
}

// 提交评价
const submitEvaluate = async () => {
  if (!canSubmit.value || loading.value) return
  
  loading.value = true
  errorMessage.value = ''
  
  const requestData = {
    student_id: authStore.user.student_id,
    conversation_id: `eval_${Date.now()}`,
    code: formData.value.code,
    problem_description: formData.value.problem_description,
    test_points: parseTestPoints(),
    task_type: 'evaluate'
  }
  
  try {
    const response = await aiAPI.evaluate(requestData)
    result.value = response
  } catch (error) {
    errorMessage.value = error.error || '评价请求失败，请稍后重试'
    console.error('Evaluate error:', error)
  } finally {
    loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  formData.value = {
    problem_description: '',
    code: ''
  }
  testPointsText.value = ''
  result.value = null
  errorMessage.value = ''
}

// 获取分数样式类
const getScoreClass = (score) => {
  if (!score) return ''
  if (score === '优秀') return 'score-excellent'
  if (score === '合格') return 'score-good'
  if (score === '待改进') return 'score-poor'
  return ''
}
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
  margin: 0 0 12px;
  font-size: 16px;
  color: #333;
  font-weight: 600;
}

textarea {
  width: 100%;
  padding: 12px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
  box-sizing: border-box;
}

textarea:focus {
  outline: none;
  border-color: #667eea;
}

.code-input {
  font-family: 'Consolas', 'Monaco', monospace;
  background: #f5f5f5;
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

.evaluation-result {
  flex: 1;
  overflow-y: auto;
}

.eval-section {
  margin-bottom: 20px;
}

.eval-section h3 {
  margin: 0 0 8px;
  color: #333;
}

.eval-section.overall {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 8px;
  border-left: 4px solid #667eea;
}

.eval-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.eval-item {
  background: #f8f9fa;
  padding: 16px;
  border-radius: 8px;
}

.eval-label {
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.eval-score {
  font-size: 24px;
  font-weight: bold;
  margin-bottom: 8px;
}

.score-excellent {
  color: #28a745;
}

.score-good {
  color: #ffc107;
}

.score-poor {
  color: #dc3545;
}

.eval-comment {
  font-size: 13px;
  color: #666;
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
  
  .eval-grid {
    grid-template-columns: 1fr;
  }
}
</style>
