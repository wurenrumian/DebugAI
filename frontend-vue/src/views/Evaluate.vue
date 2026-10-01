<template>
  <div class="evaluate-page">
    <PageHeader
      title="AI 代码评价"
      description="提交题目与代码，AI 将从功能、逻辑、效率、规范四个维度给出评价与改进建议。"
    />

    <div class="workspace">
      <!-- 左侧：输入区域 -->
      <AppCard title="提交代码" icon="code" class="input-panel">
        <form class="input-form" @submit.prevent="submitEvaluate">
          <div class="field">
            <label class="field-label" for="eval-problem">题目描述</label>
            <textarea
              id="eval-problem"
              v-model="formData.problem_description"
              placeholder="请输入题目描述..."
              rows="4"
            ></textarea>
          </div>

          <div class="field">
            <span id="eval-code-label" class="field-label" @click="codeEditor?.focus()">代码（C/C++）</span>
            <CodeEditor
              ref="codeEditor"
              id="eval-code"
              v-model="formData.code"
              labelledby="eval-code-label"
              describedby="eval-code-hint"
              placeholder="请输入需要评价的代码（C/C++）..."
              min-height="280px"
            />
            <p id="eval-code-hint" class="field-hint">Tab 缩进；按 Esc 后再按 Tab 可离开编辑器</p>
          </div>

          <div class="field">
            <label class="field-label" for="eval-tests">测试点<span class="optional">（可选）</span></label>
            <textarea
              id="eval-tests"
              v-model="testPointsText"
              class="mono-input"
              placeholder="请输入测试点，每行一个，例如：&#10;输入: 1 2 3&#10;输出: 6&#10;状态: Accepted"
              rows="4"
            ></textarea>
          </div>

          <div class="form-actions">
            <AppButton
              type="submit"
              variant="primary"
              size="lg"
              icon="sparkles"
              :loading="loading"
              :disabled="!canSubmit"
            >
              {{ loading ? '评价中...' : '提交评价' }}
            </AppButton>
            <AppButton
              v-if="hasResult"
              size="lg"
              icon="refresh"
              @click="resetForm"
            >
              重新评价
            </AppButton>
            <span class="shortcut-hint"><kbd>Ctrl</kbd> + <kbd>Enter</kbd> 提交</span>
          </div>
        </form>
      </AppCard>

      <!-- 右侧：评价结果 -->
      <AppCard title="评价结果" icon="evaluate" class="result-panel">
        <template v-if="isStreaming" #actions>
          <span class="status-pill" role="status">
            <AppSpinner size="sm" />
            AI 正在评价
          </span>
        </template>

        <AppAlert v-if="errorMessage" tone="danger" class="result-error">{{ errorMessage }}</AppAlert>

        <AppEmpty
          v-if="!hasResult"
          icon="evaluate"
          title="暂无评价结果"
          description="在左侧输入题目描述与代码，点击“提交评价”即可获得 AI 评价。"
        />

        <!-- 流式输出中：骨架屏 + 可展开的实时输出 -->
        <div v-else-if="isStreaming" class="streaming" aria-busy="true">
          <div class="skeleton-overall">
            <AppSkeleton :lines="3" />
          </div>
          <div class="skeleton-grid">
            <div v-for="name in ['功能正确', '逻辑严谨', '算法效率', '结构规范']" :key="name" class="skeleton-card">
              <span class="skeleton-title">{{ name }}</span>
              <AppSkeleton :lines="2" line-height="10px" />
            </div>
          </div>
          <details class="live-output">
            <summary>
              查看实时输出
              <span class="live-count">{{ streamingText.length }} 字符</span>
            </summary>
            <pre class="live-text">{{ streamingText }}</pre>
          </details>
        </div>

        <!-- 结构化结果 -->
        <EvaluationResult v-else :result="result" />
      </AppCard>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useKeyboardShortcut } from '@/composables/useKeyboardShortcut'
import { useAuthStore } from '@/stores/auth'
import { aiAPI } from '@/api'
import { useAbortableRequest } from '@/composables/useAbortableRequest'
import { getErrorMessage, isAbortError } from '@/utils/errors'
import PageHeader from '@/components/ui/PageHeader.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppAlert from '@/components/ui/AppAlert.vue'
import AppEmpty from '@/components/ui/AppEmpty.vue'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import AppSkeleton from '@/components/ui/AppSkeleton.vue'
import CodeEditor from '@/components/code/CodeEditor.vue'
import EvaluationResult from '@/components/results/EvaluationResult.vue'

const authStore = useAuthStore()

// 代码编辑器实例（点击标签时聚焦）
const codeEditor = ref(null)

// 流式请求控制（组件卸载 / 重新评价时自动取消）
const streamRequest = useAbortableRequest()
let requestSeq = 0

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

// 流式状态
const isStreaming = ref(false)
const streamingText = ref('')

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
  
  const requestId = ++requestSeq
  const signal = streamRequest.start()
  loading.value = true
  errorMessage.value = ''
  isStreaming.value = true
  streamingText.value = ''
  result.value = {
    overall_evaluation: '',
    functional_correctness: { grade: '', analysis: '' },
    logical_rigor: { grade: '', analysis: '' },
    algorithm_quality: { grade: '', analysis: '' },
    structural_normativity: { grade: '', analysis: '' }
  }
  
  const requestData = {
    student_id: authStore.user.student_id,
    conversation_id: `eval_${Date.now()}`,
    code: formData.value.code,
    problem_description: formData.value.problem_description,
    test_points: parseTestPoints(),
    task_type: 'evaluate'
  }
  
  try {
    await aiAPI.evaluateStream(requestData, (chunk) => {
      if (chunk.type === 'text') {
        // 流式过程中只累积文本，不解析
        streamingText.value += chunk.content
      } else if (chunk.type === 'done') {
        // 流式完成，尝试解析最终结果
        isStreaming.value = false
        try {
          const parsed = JSON.parse(streamingText.value)
          result.value = parsed
        } catch (e) {
          console.error('Final parse error:', e)
          errorMessage.value = '解析评价结果失败'
        }
      }
    }, { signal })
  } catch (error) {
    if (requestId !== requestSeq) return
    isStreaming.value = false
    // 失败时回到空状态，避免展示空的评分卡片
    result.value = null
    if (isAbortError(error)) return
    errorMessage.value = getErrorMessage(error, '评价请求失败，请稍后重试')
    console.error('Evaluate error:', error)
  } finally {
    if (requestId === requestSeq) loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  streamRequest.cancel()
  requestSeq++
  loading.value = false
  formData.value = {
    problem_description: '',
    code: ''
  }
  testPointsText.value = ''
  result.value = null
  streamingText.value = ''
  isStreaming.value = false
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
</script>

<style scoped>
.workspace {
  display: grid;
  grid-template-columns: minmax(0, 5fr) minmax(0, 6fr);
  gap: var(--space-6);
  align-items: start;
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

.result-panel {
  position: sticky;
  top: calc(var(--nav-height) + var(--space-6));
  min-height: 480px;
}

.result-error {
  margin-bottom: var(--space-4);
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px 10px;
  font-size: var(--fs-sm);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
  border-radius: var(--radius-full);
}

.streaming {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.skeleton-overall {
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.skeleton-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--color-text-muted);
}

.live-output {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.live-output summary {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  font-size: var(--fs-sm);
  color: var(--color-text-secondary);
  cursor: pointer;
  user-select: none;
}

.live-count {
  margin-left: auto;
  font-size: var(--fs-xs);
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.live-text {
  max-height: 240px;
  margin: 0;
  padding: var(--space-3);
  overflow: auto;
  font-size: var(--fs-xs);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text-secondary);
  background: var(--code-bg);
  border-top: 1px solid var(--color-border);
  border-radius: 0 0 var(--radius-md) var(--radius-md);
}

@media (max-width: 1024px) {
  .workspace {
    grid-template-columns: minmax(0, 1fr);
  }

  .result-panel {
    position: static;
    min-height: 0;
  }
}

@media (max-width: 640px) {
  .form-actions > .btn {
    flex: 1 1 auto;
  }

  .shortcut-hint {
    display: none;
  }

  .skeleton-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
