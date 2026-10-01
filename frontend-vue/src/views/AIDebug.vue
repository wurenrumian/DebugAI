<template>
  <div class="debug-page">
    <PageHeader
      title="AI 代码调试"
      description="描述问题并提交代码，AI 会通过最多 4 轮对话引导你自己定位并修正错误。"
    />

    <div class="workspace">
      <!-- 左侧：问题描述和代码输入 -->
      <AppCard title="调试信息" icon="code" class="input-panel">
        <AppAlert v-if="currentRound > 1" tone="info" class="locked-hint">
          对话进行中，题目与代码已锁定。如需修改，请点击“新建对话”。
        </AppAlert>

        <form class="input-form" @submit.prevent="startDebug">
          <div class="field">
            <label class="field-label" for="debug-problem">问题描述</label>
            <textarea
              id="debug-problem"
              v-model="formData.problem_description"
              placeholder="请描述你遇到的问题..."
              rows="4"
              :disabled="currentRound > 1"
            ></textarea>
          </div>

          <div class="field">
            <span id="debug-code-label" class="field-label" @click="codeEditor?.focus()">代码（C/C++）</span>
            <CodeEditor
              ref="codeEditor"
              id="debug-code"
              v-model="formData.code"
              labelledby="debug-code-label"
              describedby="debug-code-hint"
              placeholder="请输入需要调试的代码..."
              :disabled="currentRound > 1"
              min-height="260px"
            />
            <p id="debug-code-hint" class="field-hint">Tab 缩进；按 Esc 后再按 Tab 可离开编辑器</p>
          </div>

          <div class="field">
            <label class="field-label" for="debug-tests">测试点<span class="optional">（可选）</span></label>
            <textarea
              id="debug-tests"
              v-model="testPointsText"
              class="mono-input"
              placeholder="请输入测试点，每行一个，例如：&#10;输入: [1, 2, 3]&#10;输出: [6]"
              rows="4"
              :disabled="currentRound > 1"
            ></textarea>
          </div>

          <div class="form-actions">
            <AppButton
              type="submit"
              :variant="currentRound > 1 ? 'secondary' : 'primary'"
              size="lg"
              :icon="currentRound > 1 ? 'send' : 'sparkles'"
              :loading="loading"
              :disabled="!canStart || isConversationClosed || currentRound > 4"
            >
              {{ loading ? '处理中...' : isConversationClosed || currentRound > 4 ? '对话已关闭' : currentRound > 1 ? '继续调试' : '开始调试' }}
            </AppButton>
            <AppButton v-if="currentRound > 1" size="lg" icon="plus" @click="resetConversation">
              新建对话
            </AppButton>
            <span class="shortcut-hint"><kbd>Ctrl</kbd> + <kbd>Enter</kbd> 提交</span>
          </div>
        </form>
      </AppCard>

      <!-- 右侧：对话 -->
      <section class="chat-panel" aria-labelledby="chat-title">
        <header class="chat-header">
          <div class="chat-heading">
            <h2 id="chat-title" class="chat-title">
              <AppIcon name="chat" :size="18" />
              调试对话
            </h2>
            <p class="chat-subtitle">
              <template v-if="currentRound <= 4">第 {{ currentRound }} / 4 轮</template>
              <template v-else>已完成</template>
              <template v-if="roundInfo && roundInfo.round_title"> · {{ roundInfo.round_title }}</template>
            </p>
          </div>
          <ol class="round-steps" aria-label="调试轮次进度">
            <li
              v-for="n in 4"
              :key="n"
              :class="['round-step', {
                done: n < currentRound,
                current: n === currentRound
              }]"
              :aria-current="n === currentRound ? 'step' : undefined"
            >
              <span class="step-dot">
                <AppIcon v-if="n < currentRound" name="check" :size="12" :stroke-width="3" />
                <template v-else>{{ n }}</template>
              </span>
              <span class="sr-only">第 {{ n }} 轮{{ n < currentRound ? '（已完成）' : n === currentRound ? '（当前）' : '' }}</span>
            </li>
          </ol>
        </header>

        <!-- 轮次说明 -->
        <div v-if="roundInfo && (roundInfo.round_description || roundInfo.next_round_hint)" class="round-description">
          <p v-if="roundInfo.round_description">{{ roundInfo.round_description }}</p>
          <p v-if="roundInfo.next_round_hint" class="round-hint">
            <AppIcon name="lightbulb" :size="14" />
            {{ roundInfo.next_round_hint }}
          </p>
        </div>

        <div ref="dialogueContent" class="dialogue-content" aria-live="polite">
          <AppEmpty
            v-if="dialogueHistory.length === 0"
            icon="chat"
            title="还没有对话"
            description="在左侧填写问题描述和代码，点击“开始调试”，AI 会先复述它对你思路的理解。"
          />

          <div
            v-for="(item, index) in dialogueHistory"
            :key="index"
            :class="['message', item.role]"
          >
            <div class="message-avatar" aria-hidden="true">
              <AppIcon :name="item.role === 'student' ? 'user' : 'sparkles'" :size="16" />
            </div>
            <div class="message-body">
              <div class="message-meta">
                <span class="message-author">{{ item.role === 'student' ? '你' : 'AI 助手' }}</span>
                <span v-if="item.round_number" class="message-round">第 {{ item.round_number }} 轮</span>
              </div>
              <div class="message-bubble">
                <!-- 学生消息：纯文本显示（不解析 HTML / Markdown） -->
                <div v-if="item.role === 'student'" class="plain-text">{{ item.content }}</div>
                <!-- AI 消息：流式时显示纯文本，完成后显示结构化组件 -->
                <div v-else-if="item.role === 'assistant' && item.isStreaming" class="streaming">
                  <span class="typing" role="status">
                    <span class="typing-dots" aria-hidden="true"><i></i><i></i><i></i></span>
                    AI 正在分析
                  </span>
                  <div v-if="item.content" class="streaming-text">{{ item.content }}</div>
                </div>
                <AIResponseDisplay
                  v-else-if="item.role === 'assistant' && !item.isStreaming"
                  :ai-response="item.ai_response || parseAIResponse(item.content)"
                  :student-response="item.student_response || ''"
                />
                <div v-else class="markdown-body" v-html="renderMarkdown(item.content)"></div>
              </div>
            </div>
          </div>
        </div>

        <!-- 底部输入区（固定在对话面板底部） -->
        <footer v-if="showStudentInput || errorMessage || (currentRound > 4 && roundInfo && roundInfo.is_completed)" class="chat-footer">
          <AppAlert
            v-if="errorMessage"
            tone="danger"
            dismissible
            class="chat-error"
            @dismiss="errorMessage = ''"
          >
            {{ errorMessage }}
          </AppAlert>

          <!-- 对话完成提示 -->
          <AppAlert v-if="currentRound > 4 && (roundInfo && roundInfo.is_completed)" tone="success">
            对话已完成！如需继续调试，请新建对话。
          </AppAlert>

          <div v-if="showStudentInput" class="student-input-area">
            <!-- 第2轮：确认思路 -->
            <template v-if="currentRound === 2">
              <p id="reply-prompt" class="input-hint">请确认 AI 对你思路的理解是否正确：</p>
              <div v-if="buttonSelection === ''" class="button-choice">
                <AppButton variant="primary" :disabled="loading" @click="handleRound2Choice('correct')">
                  理解正确
                </AppButton>
                <AppButton :disabled="loading" @click="handleRound2Choice('need_correction')">
                  需要修正思路
                </AppButton>
              </div>
              <div v-else class="composer">
                <textarea
                  v-model="studentResponse"
                  aria-labelledby="reply-prompt"
                  placeholder="请输入你对思路的修正说明..."
                  rows="3"
                ></textarea>
                <AppButton
                  variant="primary"
                  icon="send"
                  :disabled="!studentResponse.trim() || loading"
                  @click="submitStudentResponse"
                >
                  发送
                </AppButton>
              </div>
            </template>

            <!-- 第3、4轮：是否需要进一步帮助 -->
            <template v-else-if="currentRound === 3 || currentRound === 4">
              <p id="reply-prompt" class="input-hint">请选择是否需要进一步帮助：</p>
              <div v-if="buttonSelection === ''" class="button-choice">
                <AppButton variant="primary" :disabled="loading" @click="handleRound3Choice('need')">
                  需要
                </AppButton>
                <AppButton :disabled="loading" @click="handleRound3Choice('not_need')">
                  不需要
                </AppButton>
              </div>
              <div v-else class="composer">
                <textarea
                  v-model="studentResponse"
                  aria-labelledby="reply-prompt"
                  placeholder="请告诉 AI 你需要什么帮助..."
                  rows="3"
                ></textarea>
                <AppButton
                  variant="primary"
                  icon="send"
                  :disabled="!studentResponse.trim() || loading"
                  @click="submitStudentResponse"
                >
                  发送
                </AppButton>
              </div>
            </template>
          </div>
        </footer>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { useAbortableRequest } from '../composables/useAbortableRequest'
import { useAuthStore } from '../stores/auth'
import { aiAPI } from '../api'
import { renderMarkdown } from '../utils/markdown'
import { getErrorMessage, isAbortError } from '../utils/errors'
import AIResponseDisplay from '../components/AIResponseDisplay.vue'
import PageHeader from '../components/ui/PageHeader.vue'
import AppCard from '../components/ui/AppCard.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppAlert from '../components/ui/AppAlert.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'
import AppIcon from '../components/ui/AppIcon.vue'
import CodeEditor from '../components/code/CodeEditor.vue'

const authStore = useAuthStore()

// 代码编辑器实例（点击标签时聚焦）
const codeEditor = ref(null)

// 流式请求控制（组件卸载 / 新建对话时自动取消）
const streamRequest = useAbortableRequest()
let requestSeq = 0

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
  
  const requestId = ++requestSeq
  const signal = streamRequest.start()
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
  
  // 预先添加一个空的 AI 消息占位（标记为流式状态）
  const history = dialogueHistory.value
  const aiMessageIndex = history.length
  dialogueHistory.value.push({
    round_number: currentRound.value,
    role: 'assistant',
    content: '',
    ai_response: null,
    isStreaming: true
  })
  
  let fullContent = ''
  try {
    await aiAPI.debugV2Stream(requestData, (chunk) => {
      if (chunk.type === 'text') {
        fullContent += chunk.content
        // 更新占位消息的内容
        dialogueHistory.value[aiMessageIndex].content = fullContent
        try {
          // 尝试解析 JSON 以更新结构化显示
          const parsed = JSON.parse(fullContent)
          dialogueHistory.value[aiMessageIndex].ai_response = parsed
        } catch (e) {
          // 忽略解析错误
        }
        scrollToBottom()
      } else if (chunk.type === 'round_info') {
        // 处理流中可能包含的轮次信息更新
        roundInfo.value = chunk.data
      } else if (chunk.type === 'done') {
        // 流式完成，标记结束
        dialogueHistory.value[aiMessageIndex].isStreaming = false
        
        // 最终解析
        try {
          const parsed = JSON.parse(fullContent)
          dialogueHistory.value[aiMessageIndex].ai_response = parsed
          
          // 如果后端在 done chunk 中返回了额外数据（如 round_info），可以在这里处理
          if (chunk.data && chunk.data.round_info) {
            roundInfo.value = chunk.data.round_info
          }
        } catch (e) {
          console.error('Final parse error:', e)
        }
        
        // 更新轮次
        currentRound.value++
        // 检查是否还有后续输入
        showStudentInput.value = currentRound.value <= 4
        // 清空学生回复和按钮选择状态
        studentResponse.value = ''
        buttonSelection.value = ''
        
        // 重新获取下一轮的轮次信息
        if (currentRound.value <= 4) {
          fetchRoundInfo(currentRound.value)
        }
      }
    }, { signal })
  } catch (error) {
    // 对话已被重置（新建对话）时，旧请求的结果直接丢弃
    if (dialogueHistory.value !== history) return
    // 如果出错 / 被取消，移除占位消息
    if (history[aiMessageIndex] && history[aiMessageIndex].isStreaming) {
      history.splice(aiMessageIndex, 1)
    }
    if (isAbortError(error)) return
    errorMessage.value = getErrorMessage(error, '请求失败，请稍后重试')
    console.error('Debug error:', error)
  } finally {
    if (requestId === requestSeq) {
      loading.value = false
      scrollToBottom()
    }
  }
}

// 提交学生回复
const submitStudentResponse = async () => {
  if (!studentResponse.value.trim() || loading.value) return
  await startDebug()
}

// 重置对话
const resetConversation = () => {
  streamRequest.cancel()
  requestSeq++
  loading.value = false
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
  const content = dialogueContent.value
  if (content) {
    content.scrollTop = content.scrollHeight
  }
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
  // 登录态由路由守卫保证
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
.workspace {
  display: grid;
  grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
  gap: var(--space-6);
  align-items: start;
}

.locked-hint {
  margin-bottom: var(--space-4);
}

.input-form {
  display: flex;
  flex-direction: column;
}

.mono-input {
  font-family: var(--font-mono);
  font-size: var(--fs-sm);
}

.form-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-top: var(--space-5);
  padding-top: var(--space-5);
  border-top: 1px solid var(--color-border);
}

.shortcut-hint {
  margin-left: auto;
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

/* ---------- 对话面板 ---------- */
.chat-panel {
  position: sticky;
  top: calc(var(--nav-height) + var(--space-6));
  display: flex;
  flex-direction: column;
  height: calc(100vh - var(--nav-height) - var(--space-12));
  min-height: 520px;
  overflow: hidden;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xs);
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.chat-heading {
  min-width: 0;
}

.chat-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--fs-lg);
}

.chat-title .app-icon {
  color: var(--color-primary-text);
}

.chat-subtitle {
  margin-top: 2px;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.round-steps {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.round-step {
  display: flex;
  align-items: center;
}

.round-step + .round-step::before {
  content: '';
  width: 18px;
  height: 2px;
  margin: 0 4px;
  background: var(--color-border-strong);
  border-radius: 2px;
}

.round-step.done + .round-step::before,
.round-step.done + .round-step.current::before {
  background: var(--color-primary);
}

.step-dot {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--color-text-muted);
  background: var(--color-surface);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-full);
}

.round-step.done .step-dot {
  color: var(--color-on-primary);
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.round-step.current .step-dot {
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-focus-ring);
}

.round-description {
  padding: var(--space-3) var(--space-5);
  font-size: var(--fs-sm);
  line-height: 1.65;
  color: var(--color-text-secondary);
  background: var(--color-surface-2);
  border-bottom: 1px solid var(--color-border);
}

.round-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 4px;
  color: var(--color-text-muted);
}

.round-hint .app-icon {
  margin-top: 3px;
  color: var(--color-warning);
}

.dialogue-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  scroll-behavior: smooth;
}

/* 消息 */
.message {
  display: flex;
  gap: var(--space-3);
  max-width: 100%;
}

.message.student {
  flex-direction: row-reverse;
}

.message-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: var(--radius-full);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border: 1px solid var(--color-primary-soft-border);
}

.message.student .message-avatar {
  color: var(--color-text-secondary);
  background: var(--color-surface-3);
  border-color: var(--color-border);
}

.message-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  max-width: min(100%, 680px);
}

.message.assistant .message-body {
  flex: 1;
}

.message.student .message-body {
  align-items: flex-end;
  max-width: 80%;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
}

.message-author {
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
}

.message-bubble {
  min-width: 0;
  max-width: 100%;
  padding: var(--space-3) var(--space-4);
  font-size: var(--fs-base);
  line-height: 1.7;
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: 4px var(--radius-lg) var(--radius-lg) var(--radius-lg);
  word-break: break-word;
}

.message.student .message-bubble {
  color: var(--color-text);
  background: var(--color-primary-soft);
  border-color: var(--color-primary-soft-border);
  border-radius: var(--radius-lg) 4px var(--radius-lg) var(--radius-lg);
}

.streaming {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.typing {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.typing-dots {
  display: inline-flex;
  gap: 3px;
}

.typing-dots i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--color-primary-text);
  animation: typing 1.2s infinite ease-in-out;
}

.typing-dots i:nth-child(2) {
  animation-delay: 0.15s;
}

.typing-dots i:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes typing {
  0%, 80%, 100% {
    opacity: 0.25;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-2px);
  }
}

.streaming-text {
  max-height: 200px;
  overflow-y: auto;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text-muted);
}

/* 底部输入区 */
.chat-footer {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
}

.input-hint {
  margin-bottom: var(--space-2);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
}

.button-choice {
  display: flex;
  gap: var(--space-3);
}

.button-choice > .btn {
  flex: 1;
}

.composer {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
}

.composer textarea {
  flex: 1;
  min-height: 72px;
  resize: none;
}

@media (max-width: 1024px) {
  .workspace {
    grid-template-columns: minmax(0, 1fr);
  }

  .chat-panel {
    position: static;
    height: min(80vh, 760px);
    min-height: 480px;
  }
}

@media (max-width: 640px) {
  .shortcut-hint {
    display: none;
  }

  .form-actions > .btn {
    flex: 1 1 auto;
  }

  .chat-header {
    flex-wrap: wrap;
    padding: var(--space-4);
  }

  .round-description,
  .chat-footer {
    padding-left: var(--space-4);
    padding-right: var(--space-4);
  }

  .dialogue-content {
    padding: var(--space-4);
  }

  .message-avatar {
    display: none;
  }

  .message.student .message-body {
    max-width: 90%;
  }

  .composer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
