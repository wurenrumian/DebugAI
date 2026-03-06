<template>
  <div class="class-selector">
    <div class="selector-header">
      <h3>我的班级</h3>
      <button 
        v-if="isAdmin" 
        class="create-btn" 
        @click="showCreateDialog = true"
      >
        + 创建班级
      </button>
    </div>
    
    <div class="class-list" v-if="classes.length > 0">
      <div 
        v-for="cls in classes" 
        :key="cls.id"
        :class="['class-item', { active: currentClassId === cls.id }]"
        @click="selectClass(cls.id)"
      >
        {{ cls.name }}
      </div>
    </div>
    <div v-else class="empty-state">
      暂无班级
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useKeyboardShortcut } from '../../composables/useKeyboardShortcut'
import { useClassStore } from '../../stores/class'
import { useAuthStore } from '../../stores/auth'

const emit = defineEmits(['class-selected'])

const classStore = useClassStore()
const authStore = useAuthStore()

const showCreateDialog = ref(false)
const newClassName = ref('')
const loading = ref(false)

const classes = computed(() => classStore.classes)
const currentClassId = computed(() => classStore.currentClass?.id)
const isAdmin = computed(() => {
  const user = authStore.user
  return user && user.user_type === 'admin'
})

// ESC 键关闭创建对话框
useKeyboardShortcut(['escape'], () => {
  if (showCreateDialog.value) {
    showCreateDialog.value = false
  }
})

onMounted(async () => {
  await classStore.fetchMyClasses()
  // 如果有班级，自动选择第一个
  if (classStore.classes.length > 0) {
    selectClass(classStore.classes[0].id)
  }
})

const selectClass = async (classId) => {
  classStore.setCurrentClass(classId)
  await classStore.fetchMembers(classId)
  emit('class-selected', classId)
}

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
      emit('class-selected', classStore.currentClass.id)
    }
  } else {
    alert(result.error || '创建班级失败')
  }
}
</script>

<style scoped>
.class-selector {
  margin-bottom: 20px;
}

.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.selector-header h3 {
  margin: 0;
  font-size: 16px;
  color: #333;
}

.create-btn {
  padding: 6px 16px;
  background-color: #409eff;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.create-btn:hover {
  background-color: #66b1ff;
}

.class-list {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.class-item {
  padding: 8px 16px;
  background-color: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.class-item:hover {
  background-color: #ecf5ff;
  border-color: #409eff;
}

.class-item.active {
  background-color: #409eff;
  color: white;
  border-color: #409eff;
}

.empty-state {
  padding: 20px;
  text-align: center;
  color: #909399;
  background-color: #f5f7fa;
  border-radius: 4px;
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
  padding: 20px;
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
