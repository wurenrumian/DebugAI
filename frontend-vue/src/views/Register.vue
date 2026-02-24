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
          <label for="username">用户名</label>
          <input 
            id="username"
            v-model="formData.username" 
            type="text" 
            placeholder="请输入用户名"
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
          <div class="hint">密码长度不少于8位</div>
        </div>
        
        <div class="form-group">
          <label for="confirmPassword">确认密码</label>
          <input 
            id="confirmPassword"
            v-model="formData.confirmPassword" 
            type="password" 
            placeholder="请再次输入密码"
            required
          />
        </div>
        
        <button type="submit" class="btn btn-primary register-btn" :disabled="loading">
          {{ loading ? '注册中...' : '注册' }}
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

const router = useRouter()
const authStore = useAuthStore()

const formData = ref({
  student_id: '',
  username: '',
  password: '',
  confirmPassword: ''
})

const loading = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

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
  
  loading.value = true
  
  try {
    const result = await authStore.register({
      student_id: formData.value.student_id,
      username: formData.value.username,
      password: formData.value.password
    })
    
    if (result.success) {
      successMessage.value = '注册成功！正在跳转到登录页面...'
      setTimeout(() => {
        router.push('/login')
      }, 1500)
    } else {
      errorMessage.value = result.error || '注册失败，请稍后重试'
    }
  } catch (error) {
    errorMessage.value = '注册失败，请检查网络连接'
  } finally {
    loading.value = false
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

.hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}

.register-btn {
  width: 100%;
  padding: 12px;
  font-size: 16px;
  margin-top: 10px;
}

.register-footer {
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
</style>
