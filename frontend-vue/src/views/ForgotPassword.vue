<template>
  <div class="forgot-password-container">
    <div class="forgot-password-box">
      <div class="forgot-password-header">
        <h1>找回密码</h1>
        <p>请输入您的注册邮箱以接收重置链接</p>
      </div>
      
      <form @submit.prevent="handleForgotPassword" class="forgot-password-form">
        <div v-if="errorMessage" class="message message-error">
          {{ errorMessage }}
        </div>
        
        <div v-if="successMessage" class="message message-success">
          {{ successMessage }}
        </div>
        
        <div class="form-group">
          <label for="email">邮箱地址</label>
          <input 
            id="email"
            v-model="email" 
            type="email" 
            placeholder="请输入注册邮箱"
            required
            :disabled="loading || emailSent"
          />
        </div>
        
        <button type="submit" class="btn btn-primary forgot-password-btn" :disabled="loading || emailSent">
          {{ loading ? '发送中...' : (emailSent ? '邮件已发送' : '发送重置邮件') }}
        </button>
        
        <div v-if="emailSent" class="resend-container">
          <p>没收到邮件？</p>
          <button type="button" @click="handleForgotPassword" :disabled="loading || countdown > 0" class="btn-link">
            {{ countdown > 0 ? `${countdown}秒后可重发` : '重新发送' }}
          </button>
        </div>
      </form>
      
      <div class="forgot-password-footer">
        <router-link to="/login" class="link">返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'

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
.forgot-password-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.forgot-password-box {
  width: 400px;
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.forgot-password-header {
  text-align: center;
  margin-bottom: 30px;
}

.forgot-password-header h1 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 8px;
}

.forgot-password-header p {
  font-size: 14px;
  color: #909399;
}

.message {
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 20px;
  font-size: 14px;
}

.message-error {
  background-color: #fef0f0;
  color: #f56c6c;
  border: 1px solid #fde2e2;
}

.message-success {
  background-color: #f0f9eb;
  color: #67c23a;
  border: 1px solid #e1f3d8;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  color: #606266;
  font-weight: 500;
}

.form-group input {
  width: 100%;
  padding: 12px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 14px;
  transition: border-color 0.3s;
  box-sizing: border-box;
}

.form-group input:focus {
  outline: none;
  border-color: #409eff;
}

.form-group input:disabled {
  background-color: #f5f7fa;
  cursor: not-allowed;
}

.btn {
  width: 100%;
  padding: 12px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
}

.btn-primary {
  background-color: #409eff;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background-color: #66b1ff;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.resend-container {
  margin-top: 20px;
  text-align: center;
  font-size: 14px;
  color: #606266;
}

.btn-link {
  background: none;
  border: none;
  color: #409eff;
  cursor: pointer;
  text-decoration: underline;
  padding: 0;
  font-size: 14px;
}

.btn-link:disabled {
  color: #909399;
  text-decoration: none;
  cursor: not-allowed;
}

.forgot-password-footer {
  text-align: center;
  margin-top: 20px;
  font-size: 14px;
}

.link {
  color: #409eff;
  text-decoration: none;
}

.link:hover {
  text-decoration: underline;
}
</style>
