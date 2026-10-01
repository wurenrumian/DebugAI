<template>
  <div class="class-selector">
    <div v-if="showHeader" class="selector-header">
      <h3>我的班级</h3>
      <AppButton
        v-if="isAdmin"
        size="sm"
        variant="primary"
        icon="plus"
        @click="showCreateDialog = true"
      >
        创建班级
      </AppButton>
    </div>

    <ul v-if="classes.length > 0" class="class-list">
      <li v-for="cls in classes" :key="cls.id">
        <button
          type="button"
          :class="['class-item', { active: currentClassId === cls.id }]"
          :aria-current="currentClassId === cls.id ? 'true' : undefined"
          @click="selectClass(cls.id)"
        >
          <span class="class-dot" aria-hidden="true"></span>
          <span class="class-label">{{ cls.name }}</span>
        </button>
      </li>
    </ul>
    <p v-else class="empty-state">暂无班级</p>

    <!-- 创建班级对话框 -->
    <AppModal
      v-if="showCreateDialog"
      title="创建班级"
      size="sm"
      @close="showCreateDialog = false"
    >
      <form :id="formId" @submit.prevent="createClass">
        <div class="field">
          <label class="field-label" :for="`${formId}-name`">班级名称</label>
          <input
            :id="`${formId}-name`"
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
          :form="formId"
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
import { useClassStore } from '../../stores/class'
import { useAuthStore } from '../../stores/auth'
import { toast } from '../../utils/toast'
import { uid } from '../../utils/uid'
import AppButton from '../ui/AppButton.vue'
import AppModal from '../ui/AppModal.vue'

defineProps({
  // 是否显示“我的班级”标题与创建按钮（ClassManage 页面在页头提供创建入口）
  showHeader: { type: Boolean, default: true }
})

const emit = defineEmits(['class-selected'])
const formId = uid('create-class')

const classStore = useClassStore()
const authStore = useAuthStore()

const showCreateDialog = ref(false)
const newClassName = ref('')
const loading = ref(false)

const classes = computed(() => classStore.classes)
const currentClassId = computed(() => classStore.currentClass?.id)
// 仅系统管理员可创建班级（后端同样校验）
const isAdmin = computed(() => authStore.isAdmin)

// 创建班级对话框的 Esc 关闭由 AppModal 处理

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
    toast.error(result.error || '创建班级失败')
  }
}
</script>

<style scoped>
.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
}

.selector-header h3 {
  font-size: var(--fs-md);
}

.class-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.class-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: 9px var(--space-3);
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
  text-align: left;
  border-radius: var(--radius-md);
  transition: background-color var(--duration-fast) var(--ease), color var(--duration-fast) var(--ease);
}

.class-item:hover {
  color: var(--color-text);
  background: var(--color-surface-hover);
}

.class-item.active {
  color: var(--color-primary-text);
  background: var(--color-primary-soft);
}

.class-dot {
  width: 8px;
  height: 8px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--color-border-strong);
}

.class-item.active .class-dot {
  background: var(--color-primary);
}

.class-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-state {
  padding: var(--space-4);
  text-align: center;
  font-size: var(--fs-sm);
  color: var(--color-text-muted);
}

@media (max-width: 900px) {
  .class-list {
    flex-direction: row;
    gap: var(--space-2);
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .class-item {
    width: auto;
    white-space: nowrap;
    border: 1px solid var(--color-border);
  }

  .class-item.active {
    border-color: var(--color-primary-soft-border);
  }

  .class-label {
    overflow: visible;
  }
}
</style>
