<template>
  <div class="profile-page">
    <section class="profile-hero" aria-labelledby="profile-name">
      <div class="avatar" aria-hidden="true">{{ avatarText }}</div>
      <div class="hero-text">
        <p class="hero-greeting">你好，</p>
        <h1 id="profile-name" class="hero-name">{{ user.username || '用户' }}</h1>
        <div class="hero-meta">
          <AppBadge :tone="user.user_type === 'admin' ? 'primary' : 'neutral'">{{ userTypeText }}</AppBadge>
          <span v-if="user.student_id" class="hero-id">学号 {{ user.student_id }}</span>
        </div>
      </div>
      <AppButton class="hero-logout" variant="secondary" icon="logout" @click="handleLogout">
        退出登录
      </AppButton>
    </section>

    <div class="profile-grid">
      <section class="features" aria-labelledby="features-title">
        <h2 id="features-title" class="section-title">开始学习</h2>
        <div class="feature-grid">
          <router-link
            v-for="item in features"
            :key="item.to"
            :to="item.to"
            class="feature-card"
          >
            <span class="feature-icon" aria-hidden="true">
              <AppIcon :name="item.icon" :size="22" />
            </span>
            <span class="feature-text">
              <span class="feature-title">{{ item.title }}</span>
              <span class="feature-desc">{{ item.desc }}</span>
            </span>
            <AppIcon name="arrow-right" :size="18" class="feature-arrow" />
          </router-link>
        </div>
      </section>

      <AppCard title="个人信息" class="info-card">
        <dl class="info-list">
          <div class="info-row">
            <dt>学号</dt>
            <dd>{{ user.student_id || '-' }}</dd>
          </div>
          <div class="info-row">
            <dt>用户名</dt>
            <dd>{{ user.username || '-' }}</dd>
          </div>
          <div class="info-row">
            <dt>账户类型</dt>
            <dd>{{ userTypeText }}</dd>
          </div>
          <div v-if="authStore.email" class="info-row info-row-email">
            <dt>邮箱</dt>
            <dd>
              <span class="email-value">{{ authStore.email }}</span>
              <AppBadge :tone="authStore.emailVerified ? 'success' : 'warning'" size="sm">
                {{ authStore.emailVerified ? '已验证' : '未验证' }}
              </AppBadge>
            </dd>
          </div>
        </dl>

        <AppButton
          v-if="authStore.email && !authStore.emailVerified"
          class="resend-btn"
          variant="secondary"
          size="sm"
          icon="mail"
          :loading="resendLoading"
          @click="handleResendVerification"
        >
          {{ resendLoading ? '发送中...' : '重新发送验证邮件' }}
        </AppButton>

        <AppAlert
          v-if="resendMessage"
          class="resend-alert"
          :tone="resendMessage.includes('成功') || resendMessage.includes('已发送') ? 'success' : 'danger'"
        >
          {{ resendMessage }}
        </AppAlert>
      </AppCard>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import AppAlert from '../components/ui/AppAlert.vue'
import AppBadge from '../components/ui/AppBadge.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppCard from '../components/ui/AppCard.vue'
import AppIcon from '../components/ui/AppIcon.vue'

const router = useRouter()
const authStore = useAuthStore()

// 功能入口
const features = [
  { to: '/ai-debug', icon: 'chat', title: 'AI 调试', desc: '多轮引导式调试，自己找到 bug' },
  { to: '/evaluate', icon: 'evaluate', title: '代码评价', desc: '从四个维度获得 AI 评价' },
  { to: '/recommend', icon: 'target', title: '题目推荐', desc: '针对薄弱点推荐练习方向' },
  { to: '/history', icon: 'clock', title: '历史记录', desc: '回看调试对话、评价与推荐' },
  { to: '/profile/classes', icon: 'users', title: '我的班级', desc: '查看班级信息、成员与学情' }
]

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
    const result = await authStore.resendVerificationEmail(authStore.email)
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
.profile-hero {
  display: flex;
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-6);
  margin-bottom: var(--space-6);
  background:
    radial-gradient(600px 200px at 0% 0%, var(--color-primary-soft), transparent 70%),
    var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xs);
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  flex-shrink: 0;
  border-radius: var(--radius-full);
  font-size: 26px;
  font-weight: var(--fw-semibold);
  color: var(--color-on-primary);
  background: var(--color-primary);
}

.hero-text {
  flex: 1;
  min-width: 0;
}

.hero-greeting {
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.hero-name {
  font-size: var(--fs-2xl);
  overflow: hidden;
  text-overflow: ellipsis;
}

.hero-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2) var(--space-3);
  margin-top: var(--space-2);
}

.hero-id {
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.profile-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  gap: var(--space-6);
  align-items: start;
}

.section-title {
  margin-bottom: var(--space-3);
  font-size: var(--fs-lg);
}

.feature-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-4);
}

.feature-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-5);
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xs);
  transition:
    border-color var(--duration) var(--ease),
    box-shadow var(--duration) var(--ease),
    transform var(--duration) var(--ease);
}

.feature-card:last-child:nth-child(odd) {
  grid-column: 1 / -1;
}

.feature-card:hover {
  text-decoration: none;
  border-color: var(--color-primary-soft-border);
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

.feature-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  border-radius: var(--radius-md);
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
}

.feature-text {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.feature-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
}

.feature-desc {
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.feature-arrow {
  color: var(--color-text-muted);
  transition: transform var(--duration) var(--ease), color var(--duration) var(--ease);
}

.feature-card:hover .feature-arrow {
  color: var(--color-primary-text);
  transform: translateX(2px);
}

.info-card {
  margin-top: calc(var(--fs-lg) * 1.35 + var(--space-3));
}

.info-list {
  display: flex;
  flex-direction: column;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--color-border);
}

.info-row:first-child {
  padding-top: 0;
}

.info-row:last-child {
  border-bottom: none;
}

.info-row dt {
  flex-shrink: 0;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

.info-row dd {
  min-width: 0;
  font-weight: var(--fw-medium);
  text-align: right;
}

.info-row-email dd {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.email-value {
  word-break: break-all;
}

.resend-btn {
  margin-top: var(--space-3);
}

.resend-alert {
  margin-top: var(--space-3);
}

@media (max-width: 1024px) {
  .profile-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .info-card {
    margin-top: 0;
  }
}

@media (max-width: 640px) {
  .profile-hero {
    flex-wrap: wrap;
    gap: var(--space-4);
    padding: var(--space-5);
  }

  .avatar {
    width: 52px;
    height: 52px;
    font-size: 22px;
  }

  .hero-name {
    font-size: var(--fs-xl);
  }

  .hero-logout {
    width: 100%;
  }

  .feature-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-3);
  }

  .feature-card {
    padding: var(--space-4);
  }
}
</style>
