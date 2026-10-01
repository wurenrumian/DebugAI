<template>
  <div class="login-page">
    <div class="auth-heading">
      <h1 class="auth-title">欢迎回来</h1>
      <p class="auth-subtitle">登录 AI 教学辅助平台，继续你的编程学习</p>
    </div>

    <form class="auth-form" @submit.prevent="handleLogin">
      <AppAlert v-if="errorMessage" tone="danger">{{ errorMessage }}</AppAlert>

      <AppAlert v-if="emailVerificationRequired" tone="warning">
        请先验证邮箱：{{ email }}
        <div class="alert-action">
          <AppButton
            variant="link"
            size="sm"
            :loading="resendLoading"
            @click="handleResendVerification"
          >
            {{ resendLoading ? '发送中...' : '重新发送验证邮件' }}
          </AppButton>
        </div>
      </AppAlert>

      <div class="field">
        <label class="field-label" for="studentId">学号</label>
        <input
          id="studentId"
          v-model="formData.student_id"
          type="text"
          placeholder="请输入学号"
          autocomplete="username"
          required
        />
      </div>

      <div class="field">
        <label class="field-label" for="password">密码</label>
        <input
          id="password"
          v-model="formData.password"
          type="password"
          placeholder="请输入密码"
          autocomplete="current-password"
          required
        />
      </div>

      <div class="auth-row-end">
        <router-link to="/forgot-password">忘记密码？</router-link>
      </div>

      <AppButton type="submit" variant="primary" size="lg" block :loading="loading">
        {{ loading ? '登录中...' : '登录' }}
      </AppButton>
    </form>

    <p class="auth-footer">
      还没有账号？<router-link to="/register">立即注册</router-link>
    </p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { toast } from '../utils/toast'
import { safeRedirectPath } from '../utils/session'
import AppAlert from '../components/ui/AppAlert.vue'
import AppButton from '../components/ui/AppButton.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formData = ref({
  student_id: '',
  password: ''
})

const loading = ref(false)
const resendLoading = ref(false)
// 登录过期被重定向过来时给出提示
const errorMessage = ref(route.query.expired ? '登录已过期，请重新登录' : '')
const emailVerificationRequired = ref(false)
const email = ref('')

// Enter 键提交登录
useKeyboardShortcut(['enter'], () => {
  if (!loading.value && formData.value.student_id && formData.value.password) {
    handleLogin()
  }
})

const handleLogin = async () => {
  errorMessage.value = ''
  emailVerificationRequired.value = false
  loading.value = true
  
  try {
    const result = await authStore.login(formData.value)
    
    if (result.success) {
      // 登录后回到原本要访问的页面（仅允许站内路径）
      router.push(safeRedirectPath(route.query.redirect))
    } else {
      // 检查是否需要邮箱验证
      if (result.error && result.error.includes('请先验证邮箱')) {
        emailVerificationRequired.value = true
        email.value = authStore.email || ''
        errorMessage.value = '请先完成邮箱验证'
      } else {
        errorMessage.value = result.error || '登录失败，请检查学号和密码'
      }
    }
  } catch (error) {
    errorMessage.value = '登录失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}

const handleResendVerification = async () => {
  resendLoading.value = true
  try {
    const result = await authStore.resendVerificationEmail(email.value || authStore.email)
    if (result.success) {
      toast.success('验证邮件已发送，请查收')
    } else {
      toast.error('发送失败：' + (result.error || '请稍后重试'))
    }
  } catch (error) {
    toast.error('发送失败，请检查网络连接')
  } finally {
    resendLoading.value = false
  }
}
</script>

<style scoped>
.alert-action {
  margin-top: var(--space-1);
}
</style>
