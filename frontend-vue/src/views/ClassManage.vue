<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📖 我的班级</h1>
    </div>
    
    <div class="content-wrapper">
      <aside class="sidebar">
        <div class="sidebar-section selector-section">
          <ClassSelector @class-selected="onClassSelected" />
        </div>
        
        <div class="sidebar-section">
          <div class="tab-nav" v-if="classStore.currentClass">
            <button
              :class="['tab-nav-btn', { active: activeTab === 'info' }]"
              @click="activeTab = 'info'"
            >
              <span class="tab-icon">📋</span>
              班级信息
            </button>
            <button
              v-if="isClassAdmin"
              :class="['tab-nav-btn', { active: activeTab === 'manage' }]"
              @click="activeTab = 'manage'"
            >
              <span class="tab-icon">👥</span>
              成员管理
            </button>
          </div>
        </div>
      </aside>
      
      <main class="main-content">
        <div v-if="classStore.currentClass" class="card content-card">
          <ClassInfoTab v-show="activeTab === 'info'" />
          <ClassManageTab v-show="activeTab === 'manage'" />
        </div>
        <div v-else class="card empty-card">
          <div class="empty-content">
            <div class="empty-icon">📚</div>
            <p>暂无班级</p>
            <p class="empty-hint">请在左侧班级选择器中创建或加入班级</p>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useClassStore } from '../stores/class'
import { useAuthStore } from '../stores/auth'
import ClassSelector from '../components/class/ClassSelector.vue'
import ClassInfoTab from '../components/class/ClassInfoTab.vue'
import ClassManageTab from '../components/class/ClassManageTab.vue'

const classStore = useClassStore()
const authStore = useAuthStore()

const activeTab = ref('info')

// 检查当前用户是否是班级管理员
const isClassAdmin = computed(() => {
  const user = authStore.getUser
  // 系统管理员可以管理任何班级
  if (user?.user_type === 'admin') return true
  
  const currentUserStudentId = user?.student_id
  if (!currentUserStudentId) return false
  const member = classStore.members.find(m => m.student_id === currentUserStudentId)
  // 班级创建者、教师、助教都可以管理
  return member && (member.role === 'teacher' || member.role === 'ta' || member.is_creator)
})

// 班级选择后的回调
const onClassSelected = (classId) => {
  activeTab.value = 'info'
}

onMounted(() => {
  // 页面加载时如果已有选中的班级，获取成员列表
  if (classStore.currentClass) {
    classStore.fetchMembers(classStore.currentClass.id)
  }
})
</script>

<style scoped>
.page-container {
  min-height: 100vh;
  background-color: #f5f7fa;
  padding: 20px;
}

.page-header {
  max-width: 1200px;
  margin: 0 auto 12px;
}

.back-link {
  display: inline-block;
  margin-bottom: 4px;
  color: #409eff;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.back-link:hover {
  color: #66b1ff;
  text-decoration: none;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.content-wrapper {
  max-width: 1200px;
  margin: 0 auto;
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.sidebar {
  width: 320px;
  flex-shrink: 0;
}

.sidebar-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  padding: 16px;
  margin-bottom: 16px;
}

.selector-section :deep(.class-selector) {
  padding: 0;
  background: transparent;
  box-shadow: none;
}

.selector-section :deep(.selector-header) {
  display: none;
}

.tab-nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tab-nav-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: #f5f7fa;
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: #606266;
  transition: all 0.2s;
  text-align: left;
}

.tab-nav-btn:hover {
  background-color: #ecf5ff;
  color: #409eff;
  border-color: #409eff;
}

.tab-nav-btn.active {
  background-color: #409eff;
  color: white;
  border-color: #409eff;
}

.tab-icon {
  font-size: 18px;
}

.main-content {
  flex: 1;
  min-width: 0;
}

.content-card {
  padding: 24px;
  min-height: 600px;
  height: 100%;
}

.empty-card {
  max-width: 800px;
  margin: 0 auto;
  padding: 80px 20px;
}

.empty-content {
  text-align: center;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 20px;
  opacity: 0.5;
}

.empty-content p {
  margin: 0;
  color: #909399;
  font-size: 18px;
}

.empty-hint {
  margin-top: 8px;
  font-size: 14px !important;
  color: #c0c4cc !important;
}
</style>

