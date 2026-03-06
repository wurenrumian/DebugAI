<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>🤖 AI 代码调试</h1>
    </div>
    
    <div class="content-wrapper ai-debug-content">
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
          :disabled="loading || !canStart || isConversationClosed || currentRound > 4"
        >
          {{ loading ? '处理中...' : isConversationClosed|currentRound>4 ? '对话已关闭' : currentRound > 1 ? '继续调试' : '开始调试' }}
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
              <span v-if="currentRound<=4" class="round-badge">第 {{ currentRound }} / 4 轮</span>
              <span v-else class="round-badge">已完成</span>
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
                <!-- 学生消息直接显示 -->
                <div v-if="item.role === 'student'" class="dialogue-text" v-html="formatContent(item.content)"></div>
                <!-- AI 消息使用专用组件解析显示 -->
                <AIResponseDisplay
                  v-else-if="item.role === 'assistant'"
                  :ai-response="item.ai_response || parseAIResponse(item.content)"
                  :student-response="item.student_response || ''"
                />
                <div v-else class="dialogue-text" v-html="formatContent(item.content)"></div>
              </div>
            </div>
            
            <div v-if="loading" class="dialogue-item assistant loading-item">
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
          
          <!-- 学生回复输入 -->
          <div v-if="showStudentInput" class="student-input-area">
            <!-- 第2轮：按钮选择 -->
            <div v-if="currentRound === 2" class="round-2-input">
              <div class="input-hint">
                <span>请确认 AI 对你思路的理解是否正确：</span>
              </div>
              <!-- 按钮选择模式 -->
              <div v-if="buttonSelection === ''" class="button-choice">
                <button
                  @click="handleRound2Choice('correct')"
                  class="btn btn-primary"
                  :disabled="loading"
                >
                  理解正确
                </button>
                <button
                  @click="handleRound2Choice('need_correction')"
                  class="btn btn-secondary"
                  :disabled="loading"
                >
                  需要修正思路
                </button>
              </div>
              <!-- 输入框模式（选择需要修正思路后显示） -->
              <div v-else class="text-input-mode">
                <textarea
                  v-model="studentResponse"
                  placeholder="请输入你对思路的修正说明..."
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
            </div>

            <!-- 第3轮：按钮选择 -->
            <div v-else-if="currentRound === 3" class="round-3-input">
              <div class="input-hint">
                <span>请选择是否需要进一步帮助：</span>
              </div>
              <!-- 按钮选择模式 -->
              <div v-if="buttonSelection === ''" class="button-choice">
                <button
                  @click="handleRound3Choice('need')"
                  class="btn btn-primary"
                  :disabled="loading"
                >
                  需要
                </button>
                <button
                  @click="handleRound3Choice('not_need')"
                  class="btn btn-secondary"
                  :disabled="loading"
                >
                  不需要
                </button>
              </div>
              <!-- 输入框模式（选择需要后显示） -->
              <div v-else class="text-input-mode">
                <textarea
                  v-model="studentResponse"
                  placeholder="请告诉 AI 你需要什么帮助..."
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
            </div>

            <!-- 第4轮：按钮选择 -->
            <div v-else-if="currentRound === 4" class="round-4-input">
              <div class="input-hint">
                <span>请选择是否需要进一步帮助：</span>
              </div>
              <!-- 按钮选择模式 -->
              <div v-if="buttonSelection === ''" class="button-choice">
                <button
                  @click="handleRound3Choice('need')"
                  class="btn btn-primary"
                  :disabled="loading"
                >
                  需要
                </button>
                <button
                  @click="handleRound3Choice('not_need')"
                  class="btn btn-secondary"
                  :disabled="loading"
                >
                  不需要
                </button>
              </div>
              <!-- 输入框模式（选择需要后显示） -->
              <div v-else class="text-input-mode">
                <textarea
                  v-model="studentResponse"
                  placeholder="请告诉 AI 你需要什么帮助..."
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
            </div>
          </div>
          
          <!-- 对话完成提示 -->
          <div v-if="currentRound > 4 && (roundInfo && roundInfo.is_completed)" class="completion-notice">
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
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'
import AIResponseDisplay from '../components/AIResponseDisplay.vue'

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

// 按钮选择状态
const buttonSelection = ref('') // '' | 'correct' | 'need_correction' | 'need' | 'not_need'

// 对话关闭状态
const isConversationClosed = ref(false)

// 加载状态
const loading = ref(false)
const errorMessage = ref('')

// 第2轮处理
const handleRound2Choice = (choice) => {
  buttonSelection.value = choice
  if (choice === 'correct') {
    // 理解正确，直接发送
    studentResponse.value = '理解正确'
    submitStudentResponse()
  }
  // need_correction 时，显示输入框让学生填写
}

// 第3轮处理
const handleRound3Choice = async (choice) => {
  buttonSelection.value = choice
  if (choice === 'not_need') {
    // 不需要帮助，关闭对话
    try {
      await aiAPI.closeConversation(conversationId.value)
      // 显示完成提示，不再发送请求
      isConversationClosed.value = true
      showStudentInput.value = false
      roundInfo.value = { is_completed: true }
    } catch (error) {
      console.error('关闭对话失败:', error)
      errorMessage.value = '关闭对话失败，请重试'
    }
  }
  // need 时，显示输入框让学生填写
}

// 关闭对话并结束
const closeConversation = async () => {
  try {
    await aiAPI.closeConversation(conversationId.value)
    isConversationClosed.value = true
    showStudentInput.value = false
    roundInfo.value = { is_completed: true }
  } catch (error) {
    console.error('关闭对话失败:', error)
    errorMessage.value = '关闭对话失败，请重试'
  }
}

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
        
        // 添加 AI 消息到对话历史，保存原始的 ai_response 数据
        dialogueHistory.value.push({
          round_number: currentRound.value,
          role: 'assistant',
          content: aiMessage.content,
          ai_response: response.ai_response || null
        })
      }
      
      // 更新轮次
      currentRound.value++
      
      // 检查是否还有后续输入
      showStudentInput.value = currentRound.value <= 4
      
      // 清空学生回复和按钮选择状态
      studentResponse.value = ''
      buttonSelection.value = ''
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
  buttonSelection.value = ''
  isConversationClosed.value = false
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

// 解析 AI 响应 JSON
const parseAIResponse = (content) => {
  if (!content) return null
  try {
    // 尝试解析 JSON
    if (typeof content === 'string') {
      const parsed = JSON.parse(content)
      return parsed
    }
    return content
  } catch (e) {
    // 如果不是 JSON，返回原始内容作为 content 字段
    return { content: content }
  }
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

// Enter 键提交（表单验证通过时）
useKeyboardShortcut(['enter'], (event) => {
  // 如果在输入框内，不处理（让输入框可以换行）
  if (event.target.tagName === 'TEXTAREA' || event.target.tagName === 'INPUT') {
    return
  }
  if (!loading.value && canStart.value && !isConversationClosed.value && currentRound.value <= 4) {
    startDebug()
  }
})

// Ctrl+Enter 键提交（针对 textarea）
useKeyboardShortcut(['ctrl+enter'], () => {
  if (!loading.value && canStart.value && !isConversationClosed.value && currentRound.value <= 4) {
    startDebug()
  }
})
</script>

<style scoped>
.ai-debug-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.left-panel .card {
  margin-bottom: 15px;
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

.loading-dots .dots {
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
  resize: none;
  margin-bottom: 10px;
}

.student-input-area .btn {
  width: 100%;
}

/* 按钮选择样式 */
.button-choice {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.button-choice .btn {
  flex: 1;
  padding: 12px;
  font-size: 14px;
}

.text-input-mode textarea {
  width: 100%;
  resize: none;
  margin-bottom: 10px;
}

.text-input-mode .btn {
  width: 100%;
}

/* 辅助按钮样式 */
.btn-secondary {
  background-color: #909399;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.btn-secondary:hover {
  background-color: #82848a;
}

.btn-secondary:disabled {
  background-color: #c0c4cc;
  cursor: not-allowed;
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
