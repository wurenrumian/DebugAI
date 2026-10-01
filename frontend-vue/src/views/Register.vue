<template>
  <div class="register-page">
    <div class="auth-heading">
      <h1 class="auth-title">创建账号</h1>
      <p class="auth-subtitle">加入 AI 教学辅助平台，使用学号注册</p>
    </div>

    <form class="auth-form" @submit.prevent="handleRegister">
      <AppAlert v-if="errorMessage" tone="danger">{{ errorMessage }}</AppAlert>

      <AppAlert v-if="successMessage" tone="success">{{ successMessage }}</AppAlert>

      <AppAlert v-if="emailSent" tone="info" title="请查收验证邮件">
        验证邮件已发送到：<strong class="email-text">{{ formData.email }}</strong>
        <br>请查收邮件并点击验证链接完成注册
        <div class="alert-action">
          <AppButton
            variant="link"
            size="sm"
            :loading="resendLoading"
            @click="resendVerification"
          >
            {{ resendLoading ? '发送中...' : '重新发送验证邮件' }}
          </AppButton>
        </div>
      </AppAlert>

      <div class="field-row">
        <div class="field">
          <label class="field-label" for="studentId">学号</label>
          <input
            id="studentId"
            v-model="formData.student_id"
            type="text"
            placeholder="请输入学号"
            autocomplete="username"
            required
            :disabled="emailSent"
          />
        </div>

        <div class="field">
          <label class="field-label" for="username">用户名</label>
          <input
            id="username"
            v-model="formData.username"
            type="text"
            placeholder="请输入用户名"
            autocomplete="nickname"
            required
            :disabled="emailSent"
          />
        </div>
      </div>

      <div class="field">
        <label class="field-label" for="password">密码</label>
        <input
          id="password"
          v-model="formData.password"
          type="password"
          placeholder="请输入密码"
          autocomplete="new-password"
          aria-describedby="password-hint"
          required
          :disabled="emailSent"
        />
        <p id="password-hint" class="field-hint">密码长度不少于6位，支持数字和字母组合</p>
      </div>

      <div class="field">
        <label class="field-label" for="confirmPassword">确认密码</label>
        <input
          id="confirmPassword"
          v-model="formData.confirmPassword"
          type="password"
          placeholder="请再次输入密码"
          autocomplete="new-password"
          required
          :disabled="emailSent"
        />
      </div>

      <div class="field">
        <label class="field-label" for="email">邮箱</label>
        <input
          id="email"
          v-model="formData.email"
          type="email"
          placeholder="请输入邮箱地址"
          autocomplete="email"
          aria-describedby="email-hint"
          required
          :disabled="emailSent"
        />
        <p id="email-hint" class="field-hint">验证邮件将发送到此邮箱，请确保邮箱有效</p>
      </div>

      <AppButton
        type="submit"
        variant="primary"
        size="lg"
        block
        :loading="loading"
        :disabled="emailSent"
      >
        {{ loading ? '发送中...' : (emailSent ? '验证邮件已发送' : '发送验证邮件') }}
      </AppButton>
    </form>

    <p class="auth-footer">
      已有账号？<router-link to="/login">立即登录</router-link>
    </p>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { toast } from '../utils/toast'
import AppAlert from '../components/ui/AppAlert.vue'
import AppButton from '../components/ui/AppButton.vue'

const router = useRouter()
const authStore = useAuthStore()

const formData = ref({
  student_id: '',
  username: '',
  password: '',
  confirmPassword: '',
  email: ''
})

const loading = ref(false)
const resendLoading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const emailSent = ref(false)

// Enter 键提交注册
useKeyboardShortcut(['enter'], () => {
  if (!loading.value && !emailSent.value && formData.value.student_id && formData.value.username && formData.value.password && formData.value.confirmPassword && formData.value.email) {
    handleRegister()
  }
})

const handleRegister = async () => {
  errorMessage.value = ''
  successMessage.value = ''
  
  // 验证密码
  if (formData.value.password !== formData.value.confirmPassword) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }
  
  // 验证密码长度
  if (formData.value.password.length < 6) {
    errorMessage.value = '密码长度不能少于6位'
    return
  }
  
  // 验证邮箱格式
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(formData.value.email)) {
    errorMessage.value = '请输入有效的邮箱地址'
    return
  }
  
  loading.value = true
  
  try {
    const result = await authStore.register({
      student_id: formData.value.student_id,
      username: formData.value.username,
      password: formData.value.password,
      email: formData.value.email
    })
    
    if (result.success) {
      emailSent.value = true
      successMessage.value = '验证邮件已发送，请查收邮件完成注册'
    } else {
      errorMessage.value = result.error || '发送失败，请稍后重试'
    }
  } catch (error) {
    errorMessage.value = '发送失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}

const resendVerification = async () => {
  resendLoading.value = true
  try {
    const result = await authStore.resendVerificationEmail({
      email: formData.value.email
    })
    if (result.success) {
      toast.success('验证邮件已重新发送，请查收')
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
.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.alert-action {
  margin-top: var(--space-1);
}

.email-text {
  word-break: break-all;
}

@media (max-width: 480px) {
  .field-row {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
