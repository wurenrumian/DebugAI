<template>
  <div class="ai-debug-container">
    <div class="ai-debug-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>🤖 AI 代码调试</h1>
    </div>
    
    <div class="ai-debug-content">
      <!-- 左侧：问题描述和代码输入 -->
      <div class="left-panel">
        <div class="card">
          <h2 class="subtitle">问题描述</h2>
          <textarea 
            v-model="formData.problem_description" 
            placeholder="请描述你遇到的问题..."
            rows="4"
            :disabled="currentRound > 1"
          ></textarea>
        </div>
        
        <div class="card">
          <h2 class="subtitle">代码</h2>
          <textarea 
            v-model="formData.code" 
            placeholder="请输入需要调试的代码..."
            rows="12"
            class="code-input"
            :disabled="currentRound > 1"
          ></textarea>
        </div>
        
        <div class="card">
          <h2 class="subtitle">测试点 (可选)</h2>
          <textarea 
            v-model="testPointsText" 
            placeholder="请输入测试点，每行一个，例如：&#10;输入: [1, 2, 3]&#10;输出: [6]"
            rows="4"
            :disabled="currentRound > 1"
          ></textarea>
        </div>
        
        <button 
          @click="startDebug" 
          class="btn btn-primary start-btn" 
          :disabled="loading || !canStart"
        >
          {{ loading ? '处理中...' : currentRound > 1 ? '继续调试' : '开始调试' }}
        </button>
        
        <button 
          v-if="currentRound > 1" 
          @click="resetConversation" 
          class="btn btn-secondary reset-btn"
        >
          新建对话
        </button>
      </div>
      
      <!-- 右侧：对话历史 -->
      <div class="right-panel">
        <div class="card dialogue-card">
          <div class="dialogue-header">
            <h2 class="subtitle">对话记录</h2>
            <div class="round-info">
              <span class="round-badge">第 {{ currentRound }} / 4 轮</span>
              <span v-if="roundInfo" class="round-title">{{ roundInfo.round_title }}</span>
            </div>
          </div>
          
          <!-- 轮次说明 -->
          <div v-if="roundInfo" class="round-description">
            <p>{{ roundInfo.round_description }}</p>
            <p class="hint" v-if="roundInfo.next_round_hint">💡 {{ roundInfo.next_round_hint }}</p>
          </div>
          
          <div class="dialogue-content" ref="dialogueContent">
            <div v-if="dialogueHistory.length === 0" class="empty-dialogue">
              <p>暂无对话记录</p>
              <p class="hint">请在左侧输入问题和代码，点击"开始调试"</p>
            </div>
            
            <div 
              v-for="(item, index) in dialogueHistory" 
              :key="index" 
              :class="['dialogue-item', item.role]"
            >
              <div class="dialogue-avatar">
                {{ item.role === 'student' ? '👤' : '🤖' }}
              </div>
              <div class="dialogue-bubble">
                <div class="dialogue-label">{{ item.role === 'student' ? '你' : 'AI 助手' }}</div>
                <div class="dialogue-text" v-html="formatContent(item.content)"></div>
              </div>
            </div>
            
            <div v-if="loading" class="dialogue-item assistant loading-item">
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
          
          <!-- 学生回复输入 -->
          <div v-if="showStudentInput" class="student-input-area">
            <div class="input-hint">
              <span v-if="currentRound === 2">请确认 AI 对你思路的理解是否正确，或提出补充说明：</span>
              <span v-else-if="currentRound === 3">请选择需要帮助的问题，或请求更详细的指导：</span>
              <span v-else-if="currentRound === 4">请告诉 AI 你需要什么帮助：</span>
            </div>
            <textarea 
              v-model="studentResponse" 
              placeholder="请输入你的回答..."
              rows="3"
            ></textarea>
            <button 
              @click="submitStudentResponse" 
              class="btn btn-primary"
              :disabled="!studentResponse.trim() || loading"
            >
              发送
            </button>
          </div>
          
          <!-- 对话完成提示 -->
          <div v-if="currentRound > 4 || (roundInfo && roundInfo.is_completed)" class="completion-notice">
            <p>🎉 对话已完成！如需继续调试，请新建对话。</p>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 错误提示 -->
    <div v-if="errorMessage" class="error-toast" @click="errorMessage = ''">
      {{ errorMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'

const authStore = useAuthStore()

// 表单数据
const formData = ref({
  problem_description: '',
  code: '',
  test_points: []
})

const testPointsText = ref('')

// 对话状态
const currentRound = ref(1)
const conversationId = ref('')
const dialogueHistory = ref([])
const studentResponse = ref('')
const showStudentInput = ref(false)
const roundInfo = ref(null)

// 加载状态
const loading = ref(false)
const errorMessage = ref('')

// 计算是否能开始调试
const canStart = computed(() => {
  return formData.value.problem_description.trim() && formData.value.code.trim()
})

// 生成会话ID
const generateConversationId = () => {
  return 'conv_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
}

// 解析测试点
const parseTestPoints = () => {
  if (!testPointsText.value.trim()) return []
  return testPointsText.value.split('\n').filter(line => line.trim())
}

// 获取轮次信息
const fetchRoundInfo = async (round, response = '') => {
  try {
    const result = await aiAPI.getRoundInfo(round, response)
    if (result && result.data) {
      roundInfo.value = result.data
    }
  } catch (error) {
    console.error('Failed to fetch round info:', error)
  }
}

// 开始调试
const startDebug = async () => {
  if (!canStart.value || loading.value) return
  
  // 如果是第一轮，初始化对话
  if (currentRound.value === 1) {
    conversationId.value = generateConversationId()
    dialogueHistory.value = []
  }
  
  loading.value = true
  errorMessage.value = ''
  
  const requestData = {
    student_id: authStore.user.student_id,
    conversation_id: conversationId.value,
    code: formData.value.code,
    problem_description: formData.value.problem_description,
    test_points: parseTestPoints(),
    current_round: currentRound.value,
    dialogue_history: dialogueHistory.value,
    student_response: currentRound.value === 1 ? '' : studentResponse.value
  }
  
  // 添加用户消息到对话历史
  if (currentRound.value > 1) {
    dialogueHistory.value.push({
      round_number: currentRound.value,
      role: 'student',
      content: studentResponse.value
    })
  }
  
  try {
    const response = await aiAPI.debugV2(requestData)
    
    if (response) {
      // 更新轮次信息
      if (response.round_info) {
        roundInfo.value = response.round_info
      }
      
      // 如果有对话记录
      if (response.dialogue_turn) {
        const aiMessage = response.dialogue_turn
        
        // 添加 AI 消息到对话历史
        dialogueHistory.value.push({
          round_number: currentRound.value,
          role: 'assistant',
          content: aiMessage.content
        })
      }
      
      // 更新轮次
      currentRound.value++
      
      // 检查是否还有后续输入
      showStudentInput.value = currentRound.value <= 4
      
      // 清空学生回复
      studentResponse.value = ''
    }
  } catch (error) {
    errorMessage.value = error.error || '请求失败，请稍后重试'
    console.error('Debug error:', error)
  } finally {
    loading.value = false
    scrollToBottom()
  }
}

// 提交学生回复
const submitStudentResponse = async () => {
  if (!studentResponse.value.trim() || loading.value) return
  await startDebug()
}

// 重置对话
const resetConversation = () => {
  currentRound.value = 1
  conversationId.value = ''
  dialogueHistory.value = []
  studentResponse.value = ''
  showStudentInput.value = false
  roundInfo.value = null
  formData.value = {
    problem_description: '',
    code: '',
    test_points: []
  }
  testPointsText.value = ''
}

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick()
  const content = document.querySelector('.dialogue-content')
  if (content) {
    content.scrollTop = content.scrollHeight
  }
}

// 格式化内容（简单的 Markdown 转换）
const formatContent = (content) => {
  if (!content) return ''
  // 转义 HTML
  let formatted = content
    .replace(/&/g, '&')
    .replace(/</g, '<')
    .replace(/>/g, '>')
  
  // 简单的代码块转换
  formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
  formatted = formatted.replace(/`([^`]+)`/g, '<code>$1</code>')
  
  // 换行转换
  formatted = formatted.replace(/\n/g, '<br>')
  
  return formatted
}

// 监听轮次变化，获取轮次信息
watch(currentRound, (newRound) => {
  if (newRound >= 1 && newRound <= 4) {
    fetchRoundInfo(newRound, studentResponse.value)
  }
})

// 获取 DOM 引用
const dialogueContent = ref(null)

onMounted(() => {
  if (!authStore.isAuthenticated) {
    window.location.href = '/login'
  }
  // 获取第一轮的初始信息
  fetchRoundInfo(1)
})
</script>

<style scoped>
.ai-debug-container {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.ai-debug-header {
  background: white;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.ai-debug-header h1 {
  font-size: 24px;
  color: #303133;
  margin-top: 10px;
}

.back-link {
  color: #409eff;
  font-size: 14px;
}

.back-link:hover {
  text-decoration: underline;
}

.ai-debug-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
}

.left-panel .card {
  margin-bottom: 15px;
}

.left-panel textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  resize: vertical;
  font-family: inherit;
}

.left-panel textarea:focus {
  border-color: #409eff;
  outline: none;
}

.left-panel textarea:disabled {
  background-color: #f5f7fa;
  cursor: not-allowed;
}

.code-input {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  background-color: #f8f9fa;
}

.start-btn {
  width: 100%;
  padding: 14px;
  font-size: 16px;
  margin-bottom: 10px;
}

.reset-btn {
  width: 100%;
  padding: 12px;
  font-size: 14px;
}

.right-panel {
  position: sticky;
  top: 20px;
  height: calc(100vh - 120px);
}

.dialogue-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.dialogue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.round-info {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.round-badge {
  background: #409eff;
  color: white;
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
}

.round-title {
  font-size: 12px;
  color: #67c23a;
  font-weight: 500;
}

.round-description {
  background: #f0f9eb;
  border: 1px solid #e1f3d8;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 15px;
}

.round-description p {
  font-size: 14px;
  color: #303133;
  line-height: 1.6;
}

.round-description .hint {
  color: #67c23a;
  font-size: 13px;
  margin-top: 8px;
}

.dialogue-content {
  flex: 1;
  overflow-y: auto;
  padding: 10px 0;
  max-height: 400px;
}

.empty-dialogue {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.empty-dialogue .hint {
  font-size: 12px;
  margin-top: 10px;
}

.dialogue-item {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.dialogue-item.student {
  flex-direction: row-reverse;
}

.dialogue-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
}

.dialogue-item.assistant .dialogue-avatar {
  background: #e6f7ff;
}

.dialogue-bubble {
  max-width: 80%;
  padding: 12px 16px;
  border-radius: 12px;
  background: #f5f7fa;
}

.dialogue-item.student .dialogue-bubble {
  background: #409eff;
  color: white;
}

.dialogue-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 5px;
}

.dialogue-item.student .dialogue-label {
  color: rgba(255, 255, 255, 0.7);
}

.dialogue-text {
  font-size: 14px;
  line-height: 1.6;
}

.dialogue-text pre {
  background: #2d2d2d;
  color: #f8f8f2;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 10px 0;
}

.dialogue-text code {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
}

.dialogue-item.student .dialogue-text code {
  background: rgba(255, 255, 255, 0.2);
}

.loading-item {
  opacity: 0.7;
}

.loading .dots {
  animation: blink 1.5s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.student-input-area {
  border-top: 1px solid #f0f0f0;
  padding-top: 15px;
  margin-top: 15px;
}

.input-hint {
  font-size: 13px;
  color: #606266;
  margin-bottom: 10px;
}

.student-input-area textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  resize: none;
  margin-bottom: 10px;
}

.student-input-area .btn {
  width: 100%;
}

.completion-notice {
  text-align: center;
  padding: 15px;
  background: #f0f9eb;
  border: 1px solid #e1f3d8;
  border-radius: 6px;
  margin-top: 15px;
}

.completion-notice p {
  color: #67c23a;
  font-size: 14px;
}

.error-toast {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: #f56c6c;
  color: white;
  padding: 12px 24px;
  border-radius: 6px;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(245, 108, 108, 0.3);
}

@media (max-width: 900px) {
  .ai-debug-content {
    grid-template-columns: 1fr;
  }
  
  .right-panel {
    position: static;
    height: auto;
  }
  
  .dialogue-content {
    max-height: 400px;
  }
}
</style>
