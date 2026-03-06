<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📝 AI 代码评价</h1>
    </div>
    
    <div class="content-wrapper ai-debug-content">
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
                <div class="eval-analysis">{{ result.functional_correctness?.analysis }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">🔍 逻辑严谨</div>
                <div class="eval-score" :class="getScoreClass(result.logical_rigor?.grade)">
                  {{ result.logical_rigor?.grade || 'N/A' }}
                </div>
                <div class="eval-analysis">{{ result.logical_rigor?.analysis }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">⚡ 算法效率</div>
                <div class="eval-score" :class="getScoreClass(result.algorithm_quality?.grade)">
                  {{ result.algorithm_quality?.grade || 'N/A' }}
                </div>
                <div class="eval-analysis">{{ result.algorithm_quality?.analysis }}</div>
              </div>
              
              <div class="eval-item">
                <div class="eval-label">📐 结构规范</div>
                <div class="eval-score" :class="getScoreClass(result.structural_normativity?.grade)">
                  {{ result.structural_normativity?.grade || 'N/A' }}
                </div>
                <div class="eval-analysis">{{ result.structural_normativity?.analysis }}</div>
              </div>
            </div>
          </div>
          
          <div v-if="loading" class="loading-item">
            <div class="dialogue-avatar">🤖</div>
            <div class="dialogue-bubble">
              <div class="dialogue-label">AI 助手</div>
              <div class="dialogue-text loading-dots">
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
import { useKeyboardShortcut } from '@/composables/useKeyboardShortcut'
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

// Enter 键提交（表单验证通过时）
useKeyboardShortcut(['enter'], (event) => {
  // 如果在输入框内，不处理（让输入框可以换行）
  if (event.target.tagName === 'TEXTAREA' || event.target.tagName === 'INPUT') {
    return
  }
  if (!loading.value && canSubmit.value) {
    submitEvaluate()
  }
})

// Ctrl+Enter 键提交（针对 textarea）
useKeyboardShortcut(['ctrl+enter'], () => {
  if (!loading.value && canSubmit.value) {
    submitEvaluate()
  }
})

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
  margin-bottom: 16px;
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

.eval-analysis {
  font-size: 13px;
  color: #666;
  line-height: 1.6;
  background: #fff;
  padding: 10px;
  border-radius: 6px;
  border: 1px solid #e9ecef;
  white-space: pre-wrap;
  word-break: break-word;
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

@media (max-width: 900px) {
  .ai-debug-content {
    grid-template-columns: 1fr;
  }

  .eval-grid {
    grid-template-columns: 1fr;
  }
}
</style>
