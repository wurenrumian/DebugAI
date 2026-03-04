<template>
  <div class="login-container">
    <div class="login-box">
      <div class="login-header">
        <h1>AI 教学辅助平台</h1>
        <p>智能代码调试助手</p>
      </div>
      
      <form @submit.prevent="handleLogin" class="login-form">
        <div v-if="errorMessage" class="message message-error">
          {{ errorMessage }}
        </div>
        
        <div v-if="emailVerificationRequired" class="message message-warning">
          请先验证邮箱：{{ email }}
          <button type="button" @click="handleResendVerification" :disabled="resendLoading" class="btn-link">
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
          />
        </div>
        
        <button type="submit" class="btn btn-primary login-btn" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
      
      <div class="login-footer">
        <span>还没有账号？</span>
        <router-link to="/register" class="link">立即注册</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const formData = ref({
  student_id: '',
  password: ''
})

const loading = ref(false)
const resendLoading = ref(false)
const errorMessage = ref('')
const emailVerificationRequired = ref(false)
const email = ref('')

const handleLogin = async () => {
  errorMessage.value = ''
  emailVerificationRequired.value = false
  loading.value = true
  
  try {
    const result = await authStore.login(formData.value)
    
    if (result.success) {
      router.push('/profile')
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
    const result = await authStore.resendVerificationEmail()
    if (result.success) {
      alert('验证邮件已发送，请查收')
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
.login-container {
  min-height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-box {
  width: 400px;
  background: white;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.login-header h1 {
  font-size: 24px;
  color: #303133;
  margin-bottom: 8px;
}

.login-header p {
  font-size: 14px;
  color: #909399;
}

.login-form {
  margin-bottom: 20px;
}

.login-btn {
  width: 100%;
  padding: 12px;
  font-size: 16px;
  margin-top: 10px;
}

.login-footer {
  text-align: center;
  font-size: 14px;
  color: #606266;
}

.link {
  color: #409eff;
  margin-left: 5px;
}

.link:hover {
  text-decoration: underline;
}

.message-warning {
  background-color: #fff3cd;
  border: 1px solid #ffeaa7;
  color: #856404;
  padding: 12px;
  border-radius: 4px;
  margin-bottom: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
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
  color: #c0c4cc;
  cursor: not-allowed;
}
</style>
