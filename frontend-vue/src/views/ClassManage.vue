<template>
  <div class="class-manage-page">
    <div class="page-header">
      <router-link to="/profile" class="back-btn">← 返回个人主页</router-link>
      <h1>班级管理</h1>
    </div>
    
    <div class="page-content">
      <ClassSelector @class-selected="onClassSelected" />
      
      <div v-if="classStore.currentClass" class="class-tabs">
        <div class="tab-headers">
          <button 
            :class="['tab-btn', { active: activeTab === 'info' }]"
            @click="activeTab = 'info'"
          >
            班级信息
          </button>
          <button 
            v-if="isClassAdmin"
            :class="['tab-btn', { active: activeTab === 'manage' }]"
            @click="activeTab = 'manage'"
          >
            成员管理
          </button>
        </div>
        
        <div class="tab-content">
          <ClassInfoTab v-show="activeTab === 'info'" />
          <ClassManageTab v-show="activeTab === 'manage'" />
        </div>
      </div>
      <div v-else class="no-class">
        <p>暂无班级，请在班级选择器中创建或加入班级</p>
      </div>
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
.class-manage-page {
  min-height: 100vh;
  background-color: #f5f7fa;
  padding: 20px;
}

.page-header {
  max-width: 800px;
  margin: 0 auto 20px;
}

.back-btn {
  display: inline-block;
  margin-bottom: 10px;
  color: #409eff;
  text-decoration: none;
  font-size: 14px;
}

.back-btn:hover {
  text-decoration: underline;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  color: #303133;
}

.page-content {
  max-width: 800px;
  margin: 0 auto;
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.class-tabs {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
  margin-top: 20px;
}

.tab-headers {
  display: flex;
  background-color: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
}

.tab-btn {
  padding: 12px 20px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: #409eff;
}

.tab-btn.active {
  color: #409eff;
  background-color: white;
  border-bottom: 2px solid #409eff;
}

.tab-content {
  padding: 16px;
  background-color: white;
}

.no-class {
  padding: 40px 20px;
  text-align: center;
  color: #909399;
  background-color: #f5f7fa;
  border-radius: 4px;
  margin-top: 20px;
}
</style>
