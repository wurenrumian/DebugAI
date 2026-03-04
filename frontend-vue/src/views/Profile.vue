<template>
  <div class="profile-container">
    <div class="profile-header">
      <div class="header-content">
        <div class="avatar">
          <span>{{ avatarText }}</span>
        </div>
        <div class="user-info">
          <h1>{{ user.username || '用户' }}</h1>
          <p class="user-type">{{ userTypeText }}</p>
        </div>
      </div>
    </div>
    
    <div class="profile-content">
      <div class="card profile-card">
        <h2 class="subtitle">个人信息</h2>
        
        <div class="info-item">
          <span class="label">学号</span>
          <span class="value">{{ user.student_id || '-' }}</span>
        </div>
        
        <div class="info-item">
          <span class="label">用户名</span>
          <span class="value">{{ user.username || '-' }}</span>
        </div>
        
        <div class="info-item">
          <span class="label">账户类型</span>
          <span class="value">{{ userTypeText }}</span>
        </div>
        
        <div class="info-item" v-if="authStore.email">
          <span class="label">邮箱</span>
          <span class="value">{{ authStore.email }}</span>
          <span class="status-badge" :class="authStore.emailVerified ? 'verified' : 'unverified'">
            {{ authStore.emailVerified ? '已验证' : '未验证' }}
          </span>
          <button
            v-if="!authStore.emailVerified"
            @click="handleResendVerification"
            :disabled="resendLoading"
            class="btn-resend"
          >
            {{ resendLoading ? '发送中...' : '重新发送验证邮件' }}
          </button>
        </div>
      </div>
      
      <div v-if="resendMessage" class="message" :class="resendMessage.includes('成功') ? 'message-success' : 'message-error'">
        {{ resendMessage }}
      </div>
      

      
      <div class="card actions-card">
        <h2 class="subtitle">AI 功能</h2>
        
        <div class="action-buttons">
          <router-link to="/ai-debug" class="action-btn">
            <div class="action-icon">🤖</div>
            <div class="action-text">
              <h3>AI 调试</h3>
              <p>多轮代码调试</p>
            </div>
          </router-link>
          
          <router-link to="/evaluate" class="action-btn">
            <div class="action-icon">📝</div>
            <div class="action-text">
              <h3>代码评价</h3>
              <p>AI 代码评价打分</p>
            </div>
          </router-link>
          
          <router-link to="/recommend" class="action-btn">
            <div class="action-icon">📚</div>
            <div class="action-text">
              <h3>题目推荐</h3>
              <p>智能推荐练习题</p>
            </div>
          </router-link>
        </div>
      </div>
      
      <div class="card history-card">
        <h2 class="subtitle">历史记录</h2>
        
        <router-link to="/history" class="action-btn full-width">
          <div class="action-icon">📜</div>
          <div class="action-text">
            <h3>查看历史</h3>
            <p>查看评分记录、对话历史、推荐题目</p>
          </div>
        </router-link>
      </div>
      
      <div class="card class-card">
        <h2 class="subtitle">班级管理</h2>
        
        <router-link to="/profile/classes" class="action-btn full-width">
          <div class="action-icon">👥</div>
          <div class="action-text">
            <h3>我的班级</h3>
            <p>管理班级成员、查看班级信息</p>
          </div>
        </router-link>
      </div>
      
      <div class="card logout-card">
        <button @click="handleLogout" class="btn btn-danger logout-btn">
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const user = ref({
  username: '',
  student_id: '',
  user_type: ''
})

const loading = ref(false)
const resendLoading = ref(false)
const resendMessage = ref('')

// 计算头像文字
const avatarText = computed(() => {
  if (user.value.username) {
    return user.value.username.charAt(0).toUpperCase()
  }
  return 'U'
})

// 计算用户类型文本
const userTypeText = computed(() => {
  return user.value.user_type === 'admin' ? '管理员' : '学生'
})

// 获取用户信息
const fetchProfile = async () => {
  loading.value = true
  try {
    const result = await authStore.fetchProfile()
    if (result.success) {
      user.value = {
        username: result.data.username || authStore.user.username,
        student_id: result.data.student_id || authStore.user.student_id,
        user_type: result.data.user_type || authStore.user.user_type
      }
    }
  } catch (error) {
    console.error('Failed to fetch profile:', error)
  } finally {
    loading.value = false
  }
}

// 重新发送验证邮件
const handleResendVerification = async () => {
  resendLoading.value = true
  resendMessage.value = ''
  try {
    const result = await authStore.resendVerificationEmail()
    if (result.success) {
      resendMessage.value = '验证邮件已发送，请查收'
    } else {
      resendMessage.value = '发送失败：' + (result.error || '请稍后重试')
    }
  } catch (error) {
    resendMessage.value = '发送失败，请检查网络连接'
  } finally {
    resendLoading.value = false
    setTimeout(() => {
      resendMessage.value = ''
    }, 5000)
  }
}

// 退出登录
const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}

onMounted(() => {
  // 从 store 获取已存储的用户信息
  if (authStore.user) {
    user.value = { ...authStore.user }
  }
  fetchProfile()
})
</script>

<style scoped>
.profile-container {
  min-height: 100vh;
  background-color: #f5f7fa;
}

.profile-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 40px 20px;
  color: white;
}

.header-content {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 20px;
}

.avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: white;
  color: #667eea;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: 600;
}

.user-info h1 {
  font-size: 28px;
  margin-bottom: 5px;
}

.user-type {
  font-size: 14px;
  opacity: 0.9;
}

.profile-content {
  max-width: 800px;
  margin: -30px auto 0;
  padding: 0 20px 40px;
}

.profile-card {
  margin-bottom: 20px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 0;
  border-bottom: 1px solid #f0f0f0;
}

.info-item:last-child {
  border-bottom: none;
}

.info-item .label {
  color: #909399;
  font-size: 14px;
}

.info-item .value {
  color: #303133;
  font-size: 14px;
  font-weight: 500;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  margin-left: 10px;
}

.status-badge.verified {
  background-color: #f0f9ff;
  color: #409eff;
  border: 1px solid #409eff;
}

.status-badge.unverified {
  background-color: #fef0f0;
  color: #f56c6c;
  border: 1px solid #f56c6c;
}

.btn-resend {
  padding: 4px 12px;
  font-size: 12px;
  background-color: #409eff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  margin-left: 10px;
}

.btn-resend:disabled {
  background-color: #c0c4cc;
  cursor: not-allowed;
}

.message {
  padding: 12px;
  border-radius: 4px;
  margin-bottom: 20px;
}

.message-success {
  background-color: #f0f9ff;
  border: 1px solid #409eff;
  color: #409eff;
}

.message-error {
  background-color: #fef0f0;
  border: 1px solid #f56c6c;
  color: #f56c6c;
}

.class-card {
  margin-bottom: 20px;
}

.actions-card, .history-card {
  margin-bottom: 20px;
}

.action-buttons {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 15px;
  margin-top: 15px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 20px;
  background: #f5f7fa;
  border-radius: 8px;
  transition: all 0.3s ease;
  text-decoration: none;
}

.action-btn:hover {
  background: #ecf5ff;
  transform: translateY(-2px);
}

.action-btn.full-width {
  grid-column: 1 / -1;
}

.action-icon {
  font-size: 32px;
}

.action-text h3 {
  font-size: 16px;
  color: #303133;
  margin-bottom: 5px;
}

.action-text p {
  font-size: 12px;
  color: #909399;
}

.logout-card {
  text-align: center;
}

.logout-btn {
  width: 100%;
  padding: 12px;
  font-size: 16px;
}

@media (max-width: 600px) {
  .action-buttons {
    grid-template-columns: 1fr;
  }
}
</style>
