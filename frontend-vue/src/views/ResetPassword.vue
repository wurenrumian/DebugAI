<template>
  <div class="reset-password-container">
    <div class="reset-password-box">
      <div class="reset-password-header">
        <h1>重置密码</h1>
        <p>请设置您的新密码</p>
      </div>
      
      <form @submit.prevent="handleResetPassword" class="reset-password-form">
        <div v-if="errorMessage" class="message message-error">
          {{ errorMessage }}
        </div>
        
        <div v-if="successMessage" class="message message-success">
          {{ successMessage }}
        </div>
        
        <div class="form-group">
          <label for="newPassword">新密码</label>
          <input 
            id="newPassword"
            v-model="newPassword" 
            type="password" 
            placeholder="请输入新密码"
            required
            :disabled="loading || success"
          />
          <div class="hint">密码长度不少于6位，支持数字和字母组合</div>
        </div>
        
        <div class="form-group">
          <label for="confirmPassword">确认新密码</label>
          <input 
            id="confirmPassword"
            v-model="confirmPassword" 
            type="password" 
            placeholder="请再次输入新密码"
            required
            :disabled="loading || success"
          />
        </div>
        
        <button type="submit" class="btn btn-primary reset-password-btn" :disabled="loading || success">
          {{ loading ? '提交中...' : (success ? '重置成功' : '重置密码') }}
        </button>
      </form>
      
      <div class="reset-password-footer">
        <router-link to="/login" class="link">返回登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

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

<style scoped>
.reset-password-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.reset-password-box {
  width: 400px;
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.reset-password-header {
  text-align: center;
  margin-bottom: 30px;
}

.reset-password-header h1 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 8px;
}

.reset-password-header p {
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

.hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
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

.reset-password-footer {
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
