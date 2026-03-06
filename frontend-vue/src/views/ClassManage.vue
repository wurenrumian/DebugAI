<template>
  <div class="page-container">
    <div class="page-header">
      <router-link to="/profile" class="back-link">← 返回个人主页</router-link>
      <h1>📖 我的班级</h1>
    </div>
    
    <div class="content-wrapper">
      <aside class="sidebar">
        <div class="sidebar-section selector-section">
          <!-- 管理员创建班级按钮 -->
          <div v-if="isSystemAdmin" class="admin-create-section">
            <button class="create-btn" @click="showCreateDialog = true">
              + 创建班级
            </button>
          </div>
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
              <span class="shortcut-hint">Ctrl+1</span>
            </button>
            <button
              v-if="isClassAdmin"
              :class="['tab-nav-btn', { active: activeTab === 'manage' }]"
              @click="activeTab = 'manage'"
            >
              <span class="tab-icon">👥</span>
              成员管理
              <span class="shortcut-hint">Ctrl+2</span>
            </button>
            <button
              v-if="isClassAdmin"
              :class="['tab-nav-btn', { active: activeTab === 'history' }]"
              @click="activeTab = 'history'"
            >
              <span class="tab-icon">📊</span>
              学生历史
              <span class="shortcut-hint">Ctrl+3</span>
            </button>
            <button
              v-if="isClassAdmin"
              :class="['tab-nav-btn', { active: activeTab === 'weakpoints' }]"
              @click="activeTab = 'weakpoints'"
            >
              <span class="tab-icon">🎯</span>
              班级薄弱点
              <span class="shortcut-hint">Ctrl+4</span>
            </button>
          </div>
        </div>
      </aside>
      
      <main class="main-content">
        <div v-if="classStore.currentClass" class="card content-card">
          <ClassInfoTab v-show="activeTab === 'info'" />
          <ClassManageTab v-show="activeTab === 'manage'" />
          <ClassHistoryQueryTab v-show="activeTab === 'history'" />
          <ClassWeakPointsQueryTab v-show="activeTab === 'weakpoints'" />
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

  <!-- 创建班级对话框 -->
  <div v-if="showCreateDialog" class="dialog-overlay" @click.self="showCreateDialog = false">
    <div class="dialog">
      <h3>创建班级</h3>
      <input
        v-model="newClassName"
        type="text"
        placeholder="请输入班级名称"
        @keyup.enter="createClass"
      />
      <div class="dialog-actions">
        <button class="cancel-btn" @click="showCreateDialog = false">取消</button>
        <button
          class="confirm-btn"
          @click="createClass"
          :disabled="!newClassName.trim() || loading"
        >
          {{ loading ? '创建中...' : '创建' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { useClassStore } from '../stores/class'
import { useAuthStore } from '../stores/auth'
import ClassSelector from '../components/class/ClassSelector.vue'
import ClassInfoTab from '../components/class/ClassInfoTab.vue'
import ClassManageTab from '../components/class/ClassManageTab.vue'
import ClassHistoryQueryTab from '../components/class/ClassHistoryQueryTab.vue'
import ClassWeakPointsQueryTab from '../components/class/ClassWeakPointsQueryTab.vue'

const classStore = useClassStore()
const authStore = useAuthStore()

const activeTab = ref('info')
const showCreateDialog = ref(false)
const newClassName = ref('')
const loading = ref(false)

// 检查是否是系统管理员
const isSystemAdmin = computed(() => {
  const user = authStore.user
  return user && user.user_type === 'admin'
})

// 检查当前用户是否是班级管理员
const isClassAdmin = computed(() => {
  const user = authStore.user
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

// ESC 键关闭创建对话框
useKeyboardShortcut(['escape'], () => {
  if (showCreateDialog.value) {
    showCreateDialog.value = false
  }
})

// 标签页切换快捷键
const tabs = ['info', 'manage', 'history', 'weakpoints']
useKeyboardShortcut(['ctrl+1', 'ctrl+2', 'ctrl+3', 'ctrl+4'], (event) => {
  const index = parseInt(event.key) - 1
  if (tabs[index] && isTabVisible(tabs[index])) {
    activeTab.value = tabs[index]
  }
})

// 检查标签是否可见（根据用户权限）
const isTabVisible = (tabName) => {
  if (tabName === 'info') return true
  if (tabName === 'manage' || tabName === 'history' || tabName === 'weakpoints') {
    return isClassAdmin.value
  }
  return false
}

// 创建班级
const createClass = async () => {
  if (!newClassName.value.trim() || loading.value) return
  
  loading.value = true
  const result = await classStore.createClass({ name: newClassName.value.trim() })
  loading.value = false
  
  if (result.success) {
    showCreateDialog.value = false
    newClassName.value = ''
    // 获取新班级的成员
    if (classStore.currentClass) {
      await classStore.fetchMembers(classStore.currentClass.id)
    }
  } else {
    alert(result.error || '创建班级失败')
  }
}
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

.shortcut-hint {
  margin-left: auto;
  font-size: 12px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid #e4e7ed;
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
.admin-create-section {
  margin-bottom: 12px;
}

.create-btn {
  width: 100%;
  padding: 10px 16px;
  background-color: #409eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.create-btn:hover {
  background-color: #66b1ff;
}

.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.dialog {
  background: white;
  padding: 24px;
  border-radius: 8px;
  width: 400px;
  max-width: 90%;
}

.dialog h3 {
  margin: 0 0 16px 0;
  font-size: 18px;
}

.dialog input {
  width: 100%;
  padding: 10px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.cancel-btn, .confirm-btn {
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.cancel-btn {
  background-color: #f5f7fa;
  border: 1px solid #dcdfe6;
  color: #606266;
}

.cancel-btn:hover {
  background-color: #f0f2f5;
}

.confirm-btn {
  background-color: #409eff;
  border: none;
  color: white;
}

.confirm-btn:hover:not(:disabled) {
  background-color: #66b1ff;
}

.confirm-btn:disabled {
  background-color: #a0cfff;
  cursor: not-allowed;
}
</style>

