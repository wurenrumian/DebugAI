<template>
  <div class="class-page">
    <PageHeader
      title="我的班级"
      description="查看班级信息；教师与助教可以管理成员、查询学生历史与班级薄弱点。"
    >
      <template v-if="isSystemAdmin" #actions>
        <AppButton variant="primary" icon="plus" @click="showCreateDialog = true">创建班级</AppButton>
      </template>
    </PageHeader>

    <div class="class-layout">
      <aside class="class-sidebar" aria-label="班级列表">
        <AppCard title="班级列表" icon="users" padding="sm">
          <ClassSelector :show-header="false" @class-selected="onClassSelected" />
        </AppCard>
      </aside>

      <main class="class-main">
        <AppCard v-if="classStore.currentClass" class="content-card" padding="lg">
          <template #header>
            <div class="class-heading">
              <h2 class="class-name">{{ classStore.currentClass.name }}</h2>
              <AppBadge v-if="isClassAdmin" tone="primary" size="sm">管理权限</AppBadge>
            </div>
          </template>

          <AppTabs v-model="activeTab" :tabs="tabs" label="班级功能">
            <ClassInfoTab v-show="activeTab === 'info'" />
            <ClassManageTab v-show="activeTab === 'manage'" />
            <ClassHistoryQueryTab v-show="activeTab === 'history'" />
            <ClassWeakPointsQueryTab v-show="activeTab === 'weakpoints'" />
          </AppTabs>
        </AppCard>

        <AppCard v-else>
          <AppEmpty
            icon="users"
            title="暂无班级"
            :description="isSystemAdmin ? '还没有班级，点击右上角“创建班级”开始。' : '你还没有加入任何班级，请联系任课教师将你加入班级。'"
          />
        </AppCard>
      </main>
    </div>

    <!-- 创建班级对话框 -->
    <AppModal
      v-if="showCreateDialog"
      title="创建班级"
      size="sm"
      @close="showCreateDialog = false"
    >
      <form id="create-class-form" @submit.prevent="createClass">
        <div class="field">
          <label class="field-label" for="new-class-name">班级名称</label>
          <input
            id="new-class-name"
            v-model="newClassName"
            type="text"
            placeholder="请输入班级名称"
          />
        </div>
      </form>
      <template #footer>
        <AppButton @click="showCreateDialog = false">取消</AppButton>
        <AppButton
          type="submit"
          form="create-class-form"
          variant="primary"
          :loading="loading"
          :disabled="!newClassName.trim()"
        >
          {{ loading ? '创建中...' : '创建' }}
        </AppButton>
      </template>
    </AppModal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useKeyboardShortcut } from '../composables/useKeyboardShortcut'
import { useClassStore } from '../stores/class'
import { useAuthStore } from '../stores/auth'
import { toast } from '../utils/toast'
import ClassSelector from '../components/class/ClassSelector.vue'
import ClassInfoTab from '../components/class/ClassInfoTab.vue'
import ClassManageTab from '../components/class/ClassManageTab.vue'
import ClassHistoryQueryTab from '../components/class/ClassHistoryQueryTab.vue'
import ClassWeakPointsQueryTab from '../components/class/ClassWeakPointsQueryTab.vue'
import PageHeader from '../components/ui/PageHeader.vue'
import AppCard from '../components/ui/AppCard.vue'
import AppButton from '../components/ui/AppButton.vue'
import AppBadge from '../components/ui/AppBadge.vue'
import AppEmpty from '../components/ui/AppEmpty.vue'
import AppModal from '../components/ui/AppModal.vue'
import AppTabs from '../components/ui/AppTabs.vue'

const classStore = useClassStore()
const authStore = useAuthStore()

const activeTab = ref('info')
const showCreateDialog = ref(false)
const newClassName = ref('')
const loading = ref(false)

// 检查是否是系统管理员
const isSystemAdmin = computed(() => authStore.isAdmin)

// 检查当前用户是否是班级管理员
const isClassAdmin = computed(() => {
  const user = authStore.user
  // 系统管理员可以管理任何班级
  if (authStore.isAdmin) return true
  
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

// 创建班级对话框的 Esc 关闭由 AppModal 处理

// 标签页（管理类标签仅班级管理员可见）
const tabs = computed(() => [
  { key: 'info', label: '班级信息', icon: 'file', shortcut: 'Ctrl+1' },
  { key: 'manage', label: '成员管理', icon: 'users', shortcut: 'Ctrl+2', hidden: !isClassAdmin.value },
  { key: 'history', label: '学生历史', icon: 'clock', shortcut: 'Ctrl+3', hidden: !isClassAdmin.value },
  { key: 'weakpoints', label: '班级薄弱点', icon: 'chart', shortcut: 'Ctrl+4', hidden: !isClassAdmin.value }
])

// 标签页切换快捷键
const tabKeys = ['info', 'manage', 'history', 'weakpoints']
useKeyboardShortcut(['ctrl+1', 'ctrl+2', 'ctrl+3', 'ctrl+4'], (event) => {
  const index = parseInt(event.key) - 1
  if (tabKeys[index] && isTabVisible(tabKeys[index])) {
    activeTab.value = tabKeys[index]
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
    toast.error(result.error || '创建班级失败')
  }
}
</script>

<style scoped>
.class-layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: var(--space-6);
  align-items: start;
}

.class-sidebar {
  position: sticky;
  top: calc(var(--nav-height) + var(--space-6));
}

.class-main {
  min-width: 0;
}

.class-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-3);
  min-width: 0;
}

.class-name {
  font-size: var(--fs-xl);
  word-break: break-word;
}

.content-card :deep(.card-header + .card-body) {
  padding-top: var(--space-3);
}

@media (max-width: 900px) {
  .class-layout {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-4);
  }

  .class-sidebar {
    position: static;
    min-width: 0;
  }
}
</style>
