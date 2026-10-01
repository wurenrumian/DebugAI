<template>
  <div class="reset-password-page">
    <div class="auth-heading">
      <h1 class="auth-title">重置密码</h1>
      <p class="auth-subtitle">请设置你的新密码</p>
    </div>

    <form class="auth-form" @submit.prevent="handleResetPassword">
      <AppAlert v-if="errorMessage" tone="danger">{{ errorMessage }}</AppAlert>

      <AppAlert v-if="successMessage" tone="success">{{ successMessage }}</AppAlert>

      <div class="field">
        <label class="field-label" for="newPassword">新密码</label>
        <input
          id="newPassword"
          v-model="newPassword"
          type="password"
          placeholder="请输入新密码"
          autocomplete="new-password"
          aria-describedby="new-password-hint"
          required
          :disabled="loading || success"
        />
        <p id="new-password-hint" class="field-hint">密码长度不少于6位，支持数字和字母组合</p>
      </div>

      <div class="field">
        <label class="field-label" for="confirmPassword">确认新密码</label>
        <input
          id="confirmPassword"
          v-model="confirmPassword"
          type="password"
          placeholder="请再次输入新密码"
          autocomplete="new-password"
          required
          :disabled="loading || success"
        />
      </div>

      <AppButton
        type="submit"
        variant="primary"
        size="lg"
        block
        :loading="loading"
        :disabled="success"
      >
        {{ loading ? '提交中...' : (success ? '重置成功' : '重置密码') }}
      </AppButton>
    </form>

    <p class="auth-footer">
      <router-link to="/login">返回登录</router-link>
    </p>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AppAlert from '../components/ui/AppAlert.vue'
import AppButton from '../components/ui/AppButton.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const newPassword = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const success = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const token = ref('')

onMounted(() => {
  token.value = route.query.token
  if (!token.value) {
    errorMessage.value = '重置令牌缺失，请通过邮件链接访问'
  }
})

const handleResetPassword = async () => {
  if (!token.value) {
    errorMessage.value = '重置令牌无效'
    return
  }
  
  errorMessage.value = ''
  successMessage.value = ''
  
  // 验证密码
  if (newPassword.value !== confirmPassword.value) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }
  
  // 验证密码长度
  if (newPassword.value.length < 6) {
    errorMessage.value = '密码长度不能少于6位'
    return
  }
  
  loading.value = true
  
  try {
    const result = await authStore.resetPassword({
      token: token.value,
      new_password: newPassword.value,
      confirm_password: confirmPassword.value
    })
    
    if (result.success) {
      success.value = true
      successMessage.value = '密码重置成功，3秒后自动跳转到登录页'
      setTimeout(() => {
        router.push('/login')
      }, 3000)
    } else {
      errorMessage.value = result.error || '重置失败，请稍后重试'
    }
  } catch (error) {
    errorMessage.value = '重置失败，请检查网络连接'
  } finally {
    loading.value = false
  }
}
</script>
