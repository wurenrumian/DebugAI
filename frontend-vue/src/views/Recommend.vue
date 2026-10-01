<template>
  <div class="recommend-page">
    <PageHeader
      title="AI 题目推荐"
      description="根据你在 AI 调试中暴露的薄弱知识点，推荐有针对性的练习方向。"
    />

    <div class="workspace">
      <!-- 左侧：薄弱点选择 -->
      <div class="left-column">
        <AppCard
          title="你的薄弱点"
          icon="chart"
          description="基于历史调试记录自动统计，点击选择要练习的知识点"
        >
          <!-- 筛选控件 -->
          <div class="filter-bar">
            <div class="field">
              <label class="field-label" for="rec-start">开始日期</label>
              <input id="rec-start" v-model="startDate" type="date" />
            </div>
            <div class="field">
              <label class="field-label" for="rec-end">结束日期</label>
              <input id="rec-end" v-model="endDate" type="date" />
            </div>
            <div class="field field-topk">
              <label class="field-label" for="rec-topk">显示前</label>
              <input id="rec-topk" v-model.number="topK" type="number" min="0" max="20" aria-describedby="rec-topk-hint" />
              <span id="rec-topk-hint" class="sr-only">0 表示显示全部</span>
            </div>
            <p class="filter-hint field-hint">“显示前”为 0 时显示全部薄弱点</p>
          </div>

          <AppSpinner v-if="loadingWeakPoints" block label="正在加载薄弱点..." />

          <AppEmpty
            v-else-if="userWeakPoints.length === 0"
            icon="chart"
            title="暂无薄弱点记录"
            description="完成更多 AI 调试后，系统会自动统计你的薄弱点。也可以调整上方的日期范围。"
            compact
          >
            <template #actions>
              <AppButton to="/ai-debug" size="sm" icon="chat">去 AI 调试</AppButton>
            </template>
          </AppEmpty>

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
        </AppCard>
      </div>

      <!-- 右侧：设置 + 推荐结果 -->
      <div class="right-column">
        <AppCard title="推荐设置" icon="target">
          <div class="settings">
            <p class="selected-summary" aria-live="polite">
              <template v-if="selectedWeakPoints.length > 0">
                已选择 <strong>{{ selectedWeakPoints.length }}</strong> 个薄弱点
              </template>
              <template v-else>请先在左侧选择至少一个薄弱点</template>
            </p>
            <div class="field setting-field">
              <label class="field-label" for="rec-count">推荐数量</label>
              <select id="rec-count" v-model="maxRecommendations">
                <option :value="3">3 个</option>
                <option :value="5">5 个</option>
                <option :value="8">8 个</option>
                <option :value="10">10 个</option>
              </select>
            </div>
            <div class="setting-actions">
              <AppButton
                variant="primary"
                size="lg"
                icon="sparkles"
                block
                :loading="loading"
                :disabled="!canSubmit"
                @click="submitRecommend"
              >
                {{ loading ? '推荐中...' : '获取推荐' }}
              </AppButton>
              <AppButton
                v-if="hasResult"
                size="lg"
                icon="refresh"
                block
                @click="resetForm"
              >
                重新推荐
              </AppButton>
            </div>
          </div>
        </AppCard>

        <AppCard title="推荐结果" icon="book" class="result-card">
          <template v-if="loading" #actions>
            <span class="status-pill" role="status">
              <AppSpinner size="sm" />
              AI 正在生成
            </span>
          </template>

          <AppAlert v-if="errorMessage" tone="danger" class="result-error">{{ errorMessage }}</AppAlert>

          <RecommendationList
            v-if="hasResult"
            :analysis="result.analysis"
            :recommendations="result.recommendations"
          />

          <div v-else-if="loading" class="result-loading" aria-busy="true">
            <AppSkeleton :lines="3" />
            <AppSkeleton :lines="2" />
          </div>

          <AppEmpty
            v-else
            icon="book"
            title="暂无推荐结果"
            description="选择薄弱点后点击“获取推荐”，AI 会给出练习方向与理由。"
            compact
          />
        </AppCard>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useKeyboardShortcut } from '@/composables/useKeyboardShortcut'
import { useAuthStore } from '@/stores/auth'
import { aiAPI } from '@/api'
import { useAbortableRequest } from '@/composables/useAbortableRequest'
import { getErrorMessage, isAbortError } from '@/utils/errors'
import WeakPointDisplay from '@/components/WeakPointDisplay.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppAlert from '@/components/ui/AppAlert.vue'
import AppEmpty from '@/components/ui/AppEmpty.vue'
import AppSpinner from '@/components/ui/AppSpinner.vue'
import AppSkeleton from '@/components/ui/AppSkeleton.vue'
import RecommendationList from '@/components/results/RecommendationList.vue'

const authStore = useAuthStore()

// 流式请求控制（组件卸载 / 重置时自动取消）
const streamRequest = useAbortableRequest()
let requestSeq = 0

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
  
  const requestId = ++requestSeq
  const signal = streamRequest.start()
  loading.value = true
  errorMessage.value = ''
  result.value = {
    analysis: '',
    recommendations: []
  }
  
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
  
  let fullContent = ''
  try {
    await aiAPI.recommendStream(requestData, (chunk) => {
      if (chunk.type === 'text') {
        fullContent += chunk.content
        try {
          const parsed = JSON.parse(fullContent)
          result.value = { ...result.value, ...parsed }
        } catch (e) {
          // 忽略解析错误
        }
      } else if (chunk.type === 'done') {
        try {
          const parsed = JSON.parse(fullContent)
          result.value = { ...result.value, ...parsed }
        } catch (e) {
          console.error('Final parse error:', e)
        }
      }
    }, { signal })
  } catch (error) {
    if (requestId !== requestSeq || isAbortError(error)) return
    errorMessage.value = getErrorMessage(error, '推荐请求失败，请稍后重试')
    console.error('Recommend error:', error)
  } finally {
    if (requestId === requestSeq) loading.value = false
  }
}

// 重置表单
const resetForm = () => {
  streamRequest.cancel()
  requestSeq++
  loading.value = false
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

// 页面加载时获取薄弱点
onMounted(() => {
  fetchUserWeakPoints()
})
</script>

<style scoped>
.workspace {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
  gap: var(--space-6);
  align-items: start;
}

.left-column,
.right-column {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  min-width: 0;
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 96px;
  gap: var(--space-3);
  padding: var(--space-4);
  margin-bottom: var(--space-5);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.filter-bar .field + .field {
  margin-top: 0;
}

.filter-hint {
  grid-column: 1 / -1;
}

.settings {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.selected-summary {
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.selected-summary strong {
  color: var(--color-primary-text);
}

.setting-field + .setting-actions {
  margin-top: 0;
}

.setting-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
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

.result-error {
  margin-bottom: var(--space-4);
}

.result-loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

@media (min-width: 1025px) {
  .right-column {
    position: sticky;
    top: calc(var(--nav-height) + var(--space-6));
  }
}

@media (max-width: 1024px) {
  .workspace {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 640px) {
  .filter-bar {
    grid-template-columns: 1fr 1fr;
    padding: var(--space-3);
  }

  .field-topk {
    grid-column: 1 / -1;
  }
}
</style>
