<template>
  <div class="register-container">
    <div class="register-box">
      <div class="register-header">
        <h1>用户注册</h1>
        <p>加入 AI 教学辅助平台</p>
      </div>
      
      <form @submit.prevent="handleRegister" class="register-form">
        <div v-if="errorMessage" class="message message-error">
          {{ errorMessage }}
        </div>
        
        <div v-if="successMessage" class="message message-success">
          {{ successMessage }}
        </div>
        
        <div v-if="emailSent" class="message message-info">
          验证邮件已发送到：{{ formData.email }}
          <br>请查收邮件并点击验证链接完成注册
          <br>
          <button type="button" @click="resendVerification" :disabled="resendLoading" class="btn-link">
            {{ resendLoading ? '发送中...' : '重新发送验证邮件' }}
          </button>
        </div>
        
        <div class="form-group">
          <label for="studentId">学号</label>
          <input 
            id="studentId"
            v-model="formData.student_id" 
            type="text" 
            placeholder="请输入学号"
            required
            :disabled="emailSent"
          />
        </div>
        
        <div class="form-group">
          <label for="username">用户名</label>
          <input 
            id="username"
            v-model="formData.username" 
            type="text" 
            placeholder="请输入用户名"
            required
            :disabled="emailSent"
          />
        </div>
        
        <div class="form-group">
          <label for="password">密码</label>
          <input
            id="password"
            v-model="formData.password"
            type="password"
            placeholder="请输入密码"
            required
            :disabled="emailSent"
          />
          <div class="hint">密码长度不少于6位，支持数字和字母组合</div>
        </div>
        
        <div class="form-group">
          <label for="confirmPassword">确认密码</label>
          <input
            id="confirmPassword"
            v-model="formData.confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            required
            :disabled="emailSent"
          />
        </div>
        
        <div class="form-group">
          <label for="email">邮箱</label>
          <input
            id="email"
            v-model="formData.email"
            type="email"
            placeholder="请输入邮箱地址"
            required
            :disabled="emailSent"
          />
          <div class="hint">验证邮件将发送到此邮箱，请确保邮箱有效</div>
        </div>
        
        <button type="submit" class="btn btn-primary register-btn" :disabled="loading || emailSent">
          {{ loading ? '发送中...' : (emailSent ? '验证邮件已发送' : '发送验证邮件') }}
        </button>
      </form>
      
      <div class="register-footer">
        <span>已有账号？</span>
        <router-link to="/login" class="link">立即登录</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'

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
      alert('验证邮件已重新发送，请查收')
    } else {
      alert('发送失败：' + (result.error || '请稍后重试'))
    }
  } catch (error) {
    alert('发送失败，请检查网络连接')
  } finally {
    resendLoading.value = false
  }
}
</script>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.register-box {
  width: 400px;
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.register-header {
  text-align: center;
  margin-bottom: 30px;
}

.register-header h1 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 8px;
}

.register-header p {
  font-size: 14px;
  color: #909399;
}

.register-form {
  margin-bottom: 20px;
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

.message-info {
  background-color: #ecf5ff;
  color: #409eff;
  border: 1px solid #d9ecff;
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

.btn-link {
  background: none;
  border: none;
  color: #409eff;
  cursor: pointer;
  text-decoration: underline;
  padding: 0;
  font-size: 14px;
  margin-left: 10px;
}

.btn-link:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.register-btn {
  margin-top: 10px;
}

.register-footer {
  text-align: center;
  margin-top: 20px;
  font-size: 14px;
  color: #909399;
}

.link {
  color: #409eff;
  text-decoration: none;
  margin-left: 5px;
}

.link:hover {
  text-decoration: underline;
}
</style>
