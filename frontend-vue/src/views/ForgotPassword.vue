<template>
  <div class="forgot-password-page">
    <div class="auth-heading">
      <h1 class="auth-title">找回密码</h1>
      <p class="auth-subtitle">输入注册邮箱，我们会发送密码重置链接</p>
    </div>

    <form class="auth-form" @submit.prevent="handleForgotPassword">
      <AppAlert v-if="errorMessage" tone="danger">{{ errorMessage }}</AppAlert>

      <AppAlert v-if="successMessage" tone="success">{{ successMessage }}</AppAlert>

      <div class="field">
        <label class="field-label" for="email">邮箱地址</label>
        <input
          id="email"
          v-model="email"
          type="email"
          placeholder="请输入注册邮箱"
          autocomplete="email"
          required
          :disabled="loading || emailSent"
        />
      </div>

      <AppButton
        type="submit"
        variant="primary"
        size="lg"
        block
        :loading="loading"
        :disabled="emailSent"
      >
        {{ loading ? '发送中...' : (emailSent ? '邮件已发送' : '发送重置邮件') }}
      </AppButton>

      <p v-if="emailSent" class="resend-row">
        没收到邮件？
        <AppButton
          variant="link"
          size="sm"
          :disabled="loading || countdown > 0"
          @click="handleForgotPassword"
        >
          {{ countdown > 0 ? `${countdown}秒后可重发` : '重新发送' }}
        </AppButton>
      </p>
    </form>

    <p class="auth-footer">
      想起密码了？<router-link to="/login">返回登录</router-link>
    </p>
  </div>
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import AppAlert from '../components/ui/AppAlert.vue'
import AppButton from '../components/ui/AppButton.vue'

const authStore = useAuthStore()

const email = ref('')
const loading = ref(false)
const emailSent = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const countdown = ref(0)
let timer = null

const startCountdown = () => {
  countdown.value = 60
  timer = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--
    } else {
      clearInterval(timer)
    }
  }, 1000)
}

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const handleForgotPassword = async () => {
  errorMessage.value = ''
  successMessage.value = ''
  
  // 验证邮箱格式
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(email.value)) {
    errorMessage.value = '请输入有效的邮箱地址'
    return
  }
  
  loading.value = true
  
  try {
    const result = await authStore.forgotPassword({ email: email.value })
    
    if (result.success) {
      emailSent.value = true
      successMessage.value = '重置邮件已发送，请查收您的邮箱'
      startCountdown()
    } else {
      errorMessage.value = result.error || '发送失败，请稍后重试'
    }
  } catch (error) {
    errorMessage.value = '发送失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.resend-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-1);
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}
</style>
